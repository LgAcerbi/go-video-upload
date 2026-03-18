//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	controller "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/middleware"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/routes"
	objectstorage "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/object-storage"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/postgres"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	uploadservice "github.com/LgAcerbi/go-video-upload/services/upload/internal/application/services"
)

const (
	testBucket     = "test-bucket"
	testAPIKey     = "test-api-key"
	dbName         = "upload_test"
	dbUser         = "test"
	dbPassword     = "test"
)

// recordUploadProcessPublisher implements ports.UploadProcessPublisher and records published upload IDs for assertions.
type recordUploadProcessPublisher struct {
	mu           sync.Mutex
	publishedIDs []string
}

func (r *recordUploadProcessPublisher) PublishUploadProcess(ctx context.Context, uploadID string) error {
	r.mu.Lock()
	r.publishedIDs = append(r.publishedIDs, uploadID)
	r.mu.Unlock()
	return nil
}

func setupIntegration(t *testing.T) (baseURL string, pool *pgxpool.Pool, uploadRepo ports.UploadRepository, videoRepo ports.VideoRepository, pubRecorder *recordUploadProcessPublisher, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	// Postgres with schema (init.sql next to this test file)
	initPath := "init.sql"
	pgCtr, err := postgresmod.Run(ctx,
		"postgres:16-alpine",
		postgresmod.WithDatabase(dbName),
		postgresmod.WithUsername(dbUser),
		postgresmod.WithPassword(dbPassword),
		postgresmod.WithInitScripts(initPath),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("postgres: %v", err)
	}
	dbURL, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	pool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}

	// MinIO
	minioCtr, err := minio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	if err != nil {
		t.Fatalf("minio: %v", err)
	}
	minioURL, err := minioCtr.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("minio connection string: %v", err)
	}
	// Ensure URL has scheme (PortEndpoint may return with or without)
	if !strings.HasPrefix(minioURL, "http") {
		minioURL = "http://" + minioURL
	}
	createMinIOBucket(t, ctx, minioURL)

	storage, err := objectstorage.NewMinIORepository(ctx, objectstorage.MinIOConfig{
		Endpoint:        minioURL,
		PresignEndpoint: minioURL,
		Region:          "us-east-1",
		Bucket:          testBucket,
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
	})
	if err != nil {
		t.Fatalf("minio repo: %v", err)
	}

	videoRepo = postgres.NewVideoRepository(pool)
	uploadRepo = postgres.NewUploadRepository(pool)
	uploadStepRepo := postgres.NewUploadStepRepository(pool)
	renditionRepo := postgres.NewRenditionRepository(pool)
	pubRecorder = &recordUploadProcessPublisher{}

	uploadSvc := uploadservice.NewUploadService(storage, testBucket, videoRepo, uploadRepo, uploadStepRepo, renditionRepo, pubRecorder)
	log := logger.New(&logger.Config{Service: "upload"})
	uploadController := controller.NewUploadController(uploadSvc, log, minioURL)

	r := chi.NewRouter()
	r.Use(middleware.RequireAPIKey(testAPIKey))
	routes.RegisterUploadRoutes(r, uploadController)

	srv := httptest.NewServer(r)
	baseURL = srv.URL

	cleanup = func() {
		srv.Close()
		pool.Close()
		_ = pgCtr.Terminate(ctx)
		_ = minioCtr.Terminate(ctx)
	}
	return baseURL, pool, uploadRepo, videoRepo, pubRecorder, cleanup
}

func createMinIOBucket(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()
	// Use AWS SDK to create bucket (minio accepts this)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"minioadmin", "minioadmin", "",
		)),
	)
	if err != nil {
		t.Fatalf("aws config: %v", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(testBucket)})
	if err != nil {
		var bne *types.BucketAlreadyExists
		var bo *types.BucketAlreadyOwnedByYou
		if errors.As(err, &bne) || errors.As(err, &bo) {
			return
		}
		t.Fatalf("create bucket: %v", err)
	}
}

func TestPresign_Integration(t *testing.T) {
	baseURL, _, uploadRepo, videoRepo, _, cleanup := setupIntegration(t)
	defer cleanup()

	userID := uuid.New().String()
	title := "Integration Test Video"
	body := map[string]string{"user_id": userID, "title": title}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/videos/upload/presign", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", testAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}

	var result struct {
		UploadURL string `json:"upload_url"`
		VideoID   string `json:"video_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.UploadURL == "" {
		t.Error("upload_url empty")
	}
	if result.VideoID == "" {
		t.Error("video_id empty")
	}

	ctx := context.Background()
	video, err := videoRepo.GetByID(ctx, result.VideoID)
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if video.UserID != userID {
		t.Errorf("video user_id: got %q", video.UserID)
	}
	if video.Title != title {
		t.Errorf("video title: got %q", video.Title)
	}

	upload, err := uploadRepo.GetByVideoID(ctx, result.VideoID)
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	if upload.Status != "pending" {
		t.Errorf("upload status: got %q", upload.Status)
	}
}

func TestPresignAndFinalize_Integration(t *testing.T) {
	baseURL, pool, _, _, pubRecorder, cleanup := setupIntegration(t)
	defer cleanup()

	userID := uuid.New().String()
	title := "Finalize Test Video"
	body := map[string]string{"user_id": userID, "title": title}
	bodyBytes, _ := json.Marshal(body)

	// 1. Presign
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/videos/upload/presign", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", testAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("presign request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("presign status: %d", resp.StatusCode)
	}
	var presignResult struct {
		UploadURL string `json:"upload_url"`
		VideoID   string `json:"video_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&presignResult); err != nil {
		t.Fatalf("presign decode: %v", err)
	}

	// 2. Upload file to presigned URL
	uploadBody := []byte("small test file content")
	putReq, _ := http.NewRequest(http.MethodPut, presignResult.UploadURL, bytes.NewReader(uploadBody))
	putReq.ContentLength = int64(len(uploadBody))
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("put to presigned URL: %v", err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusCreated {
		t.Fatalf("put status: %d", putResp.StatusCode)
	}

	// 3. Finalize
	finalizeReq, _ := http.NewRequest(http.MethodPost, baseURL+"/videos/"+presignResult.VideoID+"/upload/finalize", nil)
	finalizeReq.Header.Set("X-Api-Key", testAPIKey)
	finalizeResp, err := http.DefaultClient.Do(finalizeReq)
	if err != nil {
		t.Fatalf("finalize request: %v", err)
	}
	finalizeResp.Body.Close()
	if finalizeResp.StatusCode != http.StatusOK {
		t.Fatalf("finalize status: %d", finalizeResp.StatusCode)
	}

	// 4. Assert DB: upload status = processing, storage_path set
	ctx := context.Background()
	uploadRepo := postgres.NewUploadRepository(pool)
	upload, err := uploadRepo.GetByVideoID(ctx, presignResult.VideoID)
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	if upload.Status != "processing" {
		t.Errorf("upload status: got %q, want processing", upload.Status)
	}
	if upload.StoragePath == "" {
		t.Error("upload storage_path empty")
	}

	// 5. Assert outbound: upload-process message published with correct upload ID
	pubRecorder.mu.Lock()
	ids := append([]string(nil), pubRecorder.publishedIDs...)
	pubRecorder.mu.Unlock()
	if len(ids) != 1 {
		t.Fatalf("published upload IDs: got %d, want 1", len(ids))
	}
	if ids[0] != upload.ID {
		t.Errorf("published upload_id: got %q, want %q", ids[0], upload.ID)
	}
}

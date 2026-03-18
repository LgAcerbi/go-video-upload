//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	controller "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http"
	grpcserver "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/grpc"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/middleware"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/routes"
	objectstorage "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/object-storage"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/postgres"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	uploadservice "github.com/LgAcerbi/go-video-upload/services/upload/internal/application/services"
	uploadpb "github.com/LgAcerbi/go-video-upload/proto/upload"
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

func setupIntegration(t *testing.T) (baseURL string, pool *pgxpool.Pool, uploadRepo ports.UploadRepository, videoRepo ports.VideoRepository, pubRecorder *recordUploadProcessPublisher, grpcClient uploadpb.UploadStateServiceClient, cleanup func()) {
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

	// gRPC server (same uploadSvc)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("grpc listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	uploadpb.RegisterUploadStateServiceServer(grpcSrv, grpcserver.NewUploadStateController(uploadSvc))
	go func() { _ = grpcSrv.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	grpcClient = uploadpb.NewUploadStateServiceClient(conn)

	cleanup = func() {
		_ = conn.Close()
		grpcSrv.GracefulStop()
		_ = lis.Close()
		srv.Close()
		pool.Close()
		_ = pgCtr.Terminate(ctx)
		_ = minioCtr.Terminate(ctx)
	}
	return baseURL, pool, uploadRepo, videoRepo, pubRecorder, grpcClient, cleanup
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
	baseURL, _, uploadRepo, videoRepo, _, _, cleanup := setupIntegration(t)
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
	baseURL, pool, _, _, pubRecorder, _, cleanup := setupIntegration(t)
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

// presignAndGetIDs calls POST /videos/upload/presign and returns videoID and uploadID (from DB).
func presignAndGetIDs(t *testing.T, baseURL string, uploadRepo ports.UploadRepository) (videoID, uploadID string) {
	t.Helper()
	body := map[string]string{"user_id": uuid.New().String(), "title": "gRPC Test"}
	bodyBytes, _ := json.Marshal(body)
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
	var result struct {
		VideoID string `json:"video_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("presign decode: %v", err)
	}
	videoID = result.VideoID
	upload, err := uploadRepo.GetByVideoID(context.Background(), videoID)
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	return videoID, upload.ID
}

func TestGrpc_GetUploadProcessingContext_Integration(t *testing.T) {
	baseURL, _, uploadRepo, _, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	_, uploadID := presignAndGetIDs(t, baseURL, uploadRepo)

	ctx := context.Background()
	resp, err := grpcClient.GetUploadProcessingContext(ctx, &uploadpb.GetUploadProcessingContextRequest{UploadId: uploadID})
	if err != nil {
		t.Fatalf("GetUploadProcessingContext: %v", err)
	}
	if resp.UploadId != uploadID {
		t.Errorf("upload_id: got %q", resp.UploadId)
	}
	if resp.Status != "pending" {
		t.Errorf("status: got %q", resp.Status)
	}
	if resp.VideoId == "" {
		t.Error("video_id empty")
	}
}

func TestGrpc_GetUploadProcessingContext_NotFound_Integration(t *testing.T) {
	_, _, _, _, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	ctx := context.Background()
	_, err := grpcClient.GetUploadProcessingContext(ctx, &uploadpb.GetUploadProcessingContextRequest{UploadId: "00000000-0000-0000-0000-000000000000"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "upload not found") && !strings.Contains(err.Error(), "NotFound") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGrpc_CreateUploadSteps_UpdateUploadStep_UpdateUploadStatus_Integration(t *testing.T) {
	baseURL, _, uploadRepo, _, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	_, uploadID := presignAndGetIDs(t, baseURL, uploadRepo)
	ctx := context.Background()

	// CreateUploadSteps
	_, err := grpcClient.CreateUploadSteps(ctx, &uploadpb.CreateUploadStepsRequest{
		UploadId: uploadID,
		Steps:    []string{"extract_metadata", "transcode"},
	})
	if err != nil {
		t.Fatalf("CreateUploadSteps: %v", err)
	}

	// UpdateUploadStep
	_, err = grpcClient.UpdateUploadStep(ctx, &uploadpb.UpdateUploadStepRequest{
		UploadId: uploadID,
		Step:     "extract_metadata",
		Status:   "processing",
	})
	if err != nil {
		t.Fatalf("UpdateUploadStep: %v", err)
	}

	_, err = grpcClient.UpdateUploadStep(ctx, &uploadpb.UpdateUploadStepRequest{
		UploadId: uploadID,
		Step:     "extract_metadata",
		Status:   "done",
	})
	if err != nil {
		t.Fatalf("UpdateUploadStep done: %v", err)
	}

	// UpdateUploadStatus
	_, err = grpcClient.UpdateUploadStatus(ctx, &uploadpb.UpdateUploadStatusRequest{
		UploadId: uploadID,
		Status:   "finished",
	})
	if err != nil {
		t.Fatalf("UpdateUploadStatus: %v", err)
	}

	upload, err := uploadRepo.GetByID(ctx, uploadID)
	if err != nil {
		t.Fatalf("get upload: %v", err)
	}
	if upload.Status != "finished" {
		t.Errorf("upload status: got %q", upload.Status)
	}
}

func TestGrpc_UpdateVideoMetadata_UpdateVideoThumbnail_Integration(t *testing.T) {
	baseURL, _, uploadRepo, videoRepo, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	videoID, _ := presignAndGetIDs(t, baseURL, uploadRepo)
	ctx := context.Background()

	_, err := grpcClient.UpdateVideoMetadata(ctx, &uploadpb.UpdateVideoMetadataRequest{
		VideoId:     videoID,
		Format:      "mp4",
		DurationSec: 120.5,
		Status:      "processing",
	})
	if err != nil {
		t.Fatalf("UpdateVideoMetadata: %v", err)
	}

	video, err := videoRepo.GetByID(ctx, videoID)
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if video.Format != "mp4" {
		t.Errorf("format: got %q", video.Format)
	}
	if video.DurationSec == nil || *video.DurationSec != 120.5 {
		t.Errorf("duration_sec: got %v", video.DurationSec)
	}

	_, err = grpcClient.UpdateVideoThumbnail(ctx, &uploadpb.UpdateVideoThumbnailRequest{
		VideoId:               videoID,
		ThumbnailStoragePath:  "videos/" + videoID + "/thumb.jpg",
	})
	if err != nil {
		t.Fatalf("UpdateVideoThumbnail: %v", err)
	}
	video, _ = videoRepo.GetByID(ctx, videoID)
	if video.ThumbnailPath != "videos/"+videoID+"/thumb.jpg" {
		t.Errorf("thumbnail_path: got %q", video.ThumbnailPath)
	}
}

func TestGrpc_CreateRenditions_UpdateRendition_List_Integration(t *testing.T) {
	baseURL, _, uploadRepo, _, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	videoID, _ := presignAndGetIDs(t, baseURL, uploadRepo)
	ctx := context.Background()
	storagePath := "videos/" + videoID + "/original.mp4"

	_, err := grpcClient.CreateRenditions(ctx, &uploadpb.CreateRenditionsRequest{
		VideoId:             videoID,
		OriginalStoragePath: storagePath,
		OriginalWidth:       1920,
		OriginalHeight:      1080,
		TargetHeights:       []int32{720, 480},
	})
	if err != nil {
		t.Fatalf("CreateRenditions: %v", err)
	}

	resp, err := grpcClient.ListPendingRenditions(ctx, &uploadpb.ListPendingRenditionsRequest{VideoId: videoID})
	if err != nil {
		t.Fatalf("ListPendingRenditions: %v", err)
	}
	if len(resp.Renditions) != 2 {
		t.Errorf("pending renditions: got %d", len(resp.Renditions))
	}

	_, err = grpcClient.UpdateRendition(ctx, &uploadpb.UpdateRenditionRequest{
		VideoId:     videoID,
		Resolution:  "720p",
		StoragePath: "videos/" + videoID + "/720p.mp4",
		Width:       1280,
		Height:      720,
		Format:      "mp4",
	})
	if err != nil {
		t.Fatalf("UpdateRendition: %v", err)
	}

	readyResp, err := grpcClient.ListReadyRenditions(ctx, &uploadpb.ListReadyRenditionsRequest{VideoId: videoID})
	if err != nil {
		t.Fatalf("ListReadyRenditions: %v", err)
	}
	// CreateRenditions adds "original" (ready) + 720p/480p (pending). After UpdateRendition(720p) we have original + 720p ready.
	if len(readyResp.Renditions) != 2 {
		t.Errorf("ready renditions: got %d, want 2 (original + 720p)", len(readyResp.Renditions))
	}
	var has720p bool
	for _, r := range readyResp.Renditions {
		if r.Resolution == "720p" {
			has720p = true
			if r.StoragePath != "videos/"+videoID+"/720p.mp4" {
				t.Errorf("720p storage_path: got %q", r.StoragePath)
			}
			break
		}
	}
	if !has720p {
		t.Error("expected 720p in ready renditions")
	}
}

func TestGrpc_UpdateVideoPlayback_Integration(t *testing.T) {
	baseURL, _, uploadRepo, videoRepo, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	videoID, _ := presignAndGetIDs(t, baseURL, uploadRepo)
	ctx := context.Background()
	hlsPath := "videos/" + videoID + "/hls/master.m3u8"

	_, err := grpcClient.UpdateVideoPlayback(ctx, &uploadpb.UpdateVideoPlaybackRequest{
		VideoId:       videoID,
		HlsMasterPath: hlsPath,
	})
	if err != nil {
		t.Fatalf("UpdateVideoPlayback: %v", err)
	}

	video, err := videoRepo.GetByID(ctx, videoID)
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if video.HlsMasterPath != hlsPath {
		t.Errorf("hls_master_path: got %q", video.HlsMasterPath)
	}
	if video.Status != "published" {
		t.Errorf("status: got %q", video.Status)
	}
}

func TestGrpc_ExpireStaleUploads_Integration(t *testing.T) {
	_, _, _, _, _, grpcClient, cleanup := setupIntegration(t)
	defer cleanup()

	ctx := context.Background()
	resp, err := grpcClient.ExpireStaleUploads(ctx, &uploadpb.ExpireStaleUploadsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ExpireStaleUploads: %v", err)
	}
	if resp.Found < 0 || resp.Expired < 0 || resp.Skipped < 0 {
		t.Errorf("unexpected counts: found=%d expired=%d skipped=%d", resp.Found, resp.Expired, resp.Skipped)
	}
}

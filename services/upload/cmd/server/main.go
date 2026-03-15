package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	_ "github.com/LgAcerbi/go-video-upload/services/upload/docs"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/controllers"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/repositories"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/routes"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/services"
)

// @title           Upload Service API
// @version         1.0
// @description     API for uploading video files to object storage (S3 or MinIO).
// @host            localhost:8080
// @BasePath        /
func main() {

	log := logger.New(&logger.Config{Service: "upload"})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET is required")
	}

	objectStorage := strings.ToUpper(envOrDefault("OBJECT_STORAGE", "S3"))
	var storage ports.FileStorageRepository
	var err error
	switch objectStorage {
	case "MINIO":
		minioCfg := repository.MinIOConfig{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          envOrDefault("S3_REGION", "us-east-1"),
			Bucket:          bucket,
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}
		storage, err = repository.NewMinIOStorageRepository(ctx, minioCfg)
	case "S3":
		s3Cfg := repository.S3Config{
			Region: envOrDefault("S3_REGION", "us-east-1"),
			Bucket: bucket,
		}
		storage, err = repository.NewS3StorageRepository(ctx, s3Cfg)
	default:
		log.Fatal("OBJECT_STORAGE must be S3 or MINIO", "got", objectStorage)
	}
	if err != nil {
		log.Fatal("storage init failed", "error", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("database connection failed", "error", err)
	}
	defer pool.Close()

	videoRepo := repository.NewVideoRepository(pool)
	uploadRepo := repository.NewUploadRepository(pool)

	rabbitCfg := rabbitmq.ConfigFromEnv()
	rabbitConn, err := rabbitmq.Connect(rabbitCfg)
	if err != nil {
		log.Fatal("rabbitmq connection failed", "error", err)
	}
	defer rabbitConn.Close()

	uploadProcessPub := repository.NewRabbitMQUploadProcessPublisher(rabbitConn)

	uploadSvc := service.NewUploadService(storage, bucket, videoRepo, uploadRepo, uploadProcessPub)
	uploadController := controller.NewUploadController(uploadSvc, log)

	r := chi.NewRouter()
	routes.RegisterUploadRoutes(r, uploadController)
	r.Get("/docs/*", httpSwagger.WrapHandler)

	addr := ":" + port
	log.Info("upload service listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal("server failed", "error", err)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

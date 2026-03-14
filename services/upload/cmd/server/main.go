package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/controller"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/repository"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/routes"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/service"
)

func main() {
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
	var storage ports.FileStorage
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
		storage, err = repository.NewMinIOStorage(ctx, minioCfg)
	case "S3":
		s3Cfg := repository.S3Config{
			Region: envOrDefault("S3_REGION", "us-east-1"),
			Bucket: bucket,
		}
		storage, err = repository.NewS3Storage(ctx, s3Cfg)
	default:
		log.Fatalf("OBJECT_STORAGE must be S3 or MINIO, got %q", objectStorage)
	}
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	uploadSvc := service.NewUploadService(storage, bucket)
	uploadController := controller.NewUploadController(uploadSvc)

	r := chi.NewRouter()
	routes.RegisterUploadRoutes(r, uploadController)

	addr := ":" + port
	log.Printf("upload service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/proto/upload"
	_ "github.com/LgAcerbi/go-video-upload/services/upload/docs"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/grpc"
	controller "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/middleware"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/http/routes"
	objectstorage "github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/object-storage"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/adapters/postgres"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/services"
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
		minioCfg := objectstorage.MinIOConfig{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			PresignEndpoint: os.Getenv("S3_PRESIGN_ENDPOINT"),
			Region:          envOrDefault("S3_REGION", "us-east-1"),
			Bucket:          bucket,
			AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		}
		storage, err = objectstorage.NewMinIORepository(ctx, minioCfg)
	case "S3":
		s3Cfg := objectstorage.S3Config{
			Region: envOrDefault("S3_REGION", "us-east-1"),
			Bucket: bucket,
		}
		storage, err = objectstorage.NewS3Repository(ctx, s3Cfg)
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

	videoRepo := postgres.NewVideoRepository(pool)
	uploadRepo := postgres.NewUploadRepository(pool)
	uploadStepRepo := postgres.NewUploadStepRepository(pool)

	rabbitCfg := rabbitmq.ConfigFromEnv()
	rabbitConn, err := rabbitmq.Connect(rabbitCfg)
	if err != nil {
		log.Fatal("rabbitmq connection failed", "error", err)
	}
	defer rabbitConn.Close()

	uploadProcessPub := amqp.NewRabbitMQUploadProcessPublisher(rabbitConn)

	uploadSvc := service.NewUploadService(storage, bucket, videoRepo, uploadRepo, uploadStepRepo, uploadProcessPub)
	uploadController := controller.NewUploadController(uploadSvc, log)

	grpcServer := grpc.NewServer()
	upload.RegisterUploadStateServiceServer(grpcServer, grpcserver.NewUploadStateController(uploadSvc))
	reflection.Register(grpcServer)

	grpcPort := envOrDefault("GRPC_PORT", "9090")
	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("grpc listen failed", "error", err)
	}
	go func() {
		log.Info("grpc server listening", "addr", grpcLis.Addr())
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("grpc server failed", "error", err)
		}
	}()

	r := chi.NewRouter()
	if envOrDefault("ENVIRONMENT", "development") != "production" {
		r.Use(middleware.CORS([]string{"http://127.0.0.1", "http://localhost"}))
	}
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

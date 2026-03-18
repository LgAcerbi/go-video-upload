package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	grpcclient "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/adapters/grpc"
	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/adapters/ffmpeg"
	objectstorage "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/adapters/object-storage"
	amqp "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/adapters/rabbitmq"
	service "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/services"
)

func main() {
	log := logger.New(&logger.Config{Service: "thumbnail"})

	uploadTarget := os.Getenv("UPLOAD_GRPC_TARGET")
	if uploadTarget == "" {
		uploadTarget = "localhost:9090"
	}
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET is required")
	}

	ctx := context.Background()
	uploadClient, err := grpcclient.NewUploadStateClient(ctx, uploadTarget)
	if err != nil {
		log.Fatal("upload gRPC client failed", "error", err)
	}
	defer uploadClient.Close()

	rabbitCfg := rabbitmq.ConfigFromEnv()
	rabbitConn, err := rabbitmq.Connect(rabbitCfg)
	if err != nil {
		log.Fatal("rabbitmq connection failed", "error", err)
	}
	defer rabbitConn.Close()

	s3Cfg := objectstorage.S3Config{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		Region:          envOrDefault("S3_REGION", "us-east-1"),
		Bucket:          bucket,
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
	fetcher, err := objectstorage.NewVideoFileFetcher(ctx, s3Cfg)
	if err != nil {
		log.Fatal("object storage fetcher failed", "error", err)
	}
	uploader, err := objectstorage.NewThumbnailUploader(ctx, s3Cfg)
	if err != nil {
		log.Fatal("thumbnail uploader failed", "error", err)
	}

	thumbGen := ffmpeg.NewThumbnailGenerator()
	stepResultPub := amqp.NewStepResultPublisher(rabbitConn)
	svc := service.NewThumbnailService(uploadClient, fetcher, uploader, thumbGen, stepResultPub, bucket)

	metricsWriter, _ := metrics.NewWriter(metrics.WriterConfig{
		URL:    os.Getenv("INFLUXDB_URL"),
		Token:  os.Getenv("INFLUXDB_TOKEN"),
		Org:    envOrDefault("INFLUXDB_ORG", "org"),
		Bucket: envOrDefault("INFLUXDB_BUCKET", "metrics"),
	})
	if metricsWriter != nil {
		defer metricsWriter.Close()
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		if err := amqp.RunGenerateThumbnailConsumer(runCtx, rabbitConn, svc, metricsWriter, log); err != nil && runCtx.Err() == nil {
			log.Error("generate_thumbnail consumer exited", "error", err)
		}
	}()

	log.Info("thumbnail service started", "upload_target", uploadTarget)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")
	cancel()
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}


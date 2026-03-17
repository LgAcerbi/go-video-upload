package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/adapters/grpc"
	amqp "github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/services"
)

func main() {
	log := logger.New(&logger.Config{Service: "orchestrator"})

	uploadTarget := os.Getenv("UPLOAD_GRPC_TARGET")
	if uploadTarget == "" {
		uploadTarget = "localhost:9090"
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

	stepPublisher := amqp.NewStepPublisher(rabbitConn)
	svc := service.NewOrchestratorService(uploadClient, stepPublisher)

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
		if err := amqp.RunUploadProcessConsumer(runCtx, rabbitConn, svc, metricsWriter, log); err != nil && runCtx.Err() == nil {
			log.Error("upload-process consumer exited", "error", err)
		}
	}()
	go func() {
		if err := amqp.RunStepResultConsumer(runCtx, rabbitConn, svc, metricsWriter, log); err != nil && runCtx.Err() == nil {
			log.Error("upload-process-step consumer exited", "error", err)
		}
	}()

	log.Info("orchestrator started", "upload_target", uploadTarget)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")
	cancel()
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

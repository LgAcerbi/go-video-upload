package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	grpcclient "github.com/LgAcerbi/go-video-upload/services/expirer/internal/adapters/grpc"
	"github.com/LgAcerbi/go-video-upload/services/expirer/internal/application/services"
)

const tickInterval = time.Minute

func main() {
	log := logger.New(&logger.Config{Service: "expirer"})
	ctx := context.Background()

	uploadTarget := os.Getenv("UPLOAD_GRPC_TARGET")
	if uploadTarget == "" {
		uploadTarget = "localhost:9090"
	}

	limit := envInt("EXPIRER_BATCH_LIMIT", 200)
	timeout := time.Duration(envInt("EXPIRER_RUN_TIMEOUT_SEC", 20)) * time.Second

	uploadClient, err := grpcclient.NewUploadStateClient(ctx, uploadTarget)
	if err != nil {
		log.Fatal("upload gRPC client failed", "error", err)
	}
	defer uploadClient.Close()

	expirerSvc := service.NewExpirerService(uploadClient)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	runOnce := func() {
		runOnceCtx, cancel := context.WithTimeout(runCtx, timeout)
		defer cancel()

		res, err := expirerSvc.ExpireStaleUploads(runOnceCtx, limit)
		if err != nil && runOnceCtx.Err() == nil {
			log.Error("expire run failed", "error", err)
			return
		}
		log.Info("expire run completed", "found", res.Found, "expired", res.Expired, "skipped", res.Skipped)
	}

	runOnce()

	go func() {
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()

	log.Info("expirer started", "interval", tickInterval.String(), "batch_limit", limit, "timeout", timeout.String(), "upload_target", uploadTarget)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")
	cancel()
}

func envInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}


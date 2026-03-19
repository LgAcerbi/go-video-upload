package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/adapters/postgres"
	amqp "github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/application/services"
)

const defaultTickInterval = 5 * time.Second

func main() {
	log := logger.New(&logger.Config{Service: "outbox-dispatcher"})
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("database connection failed", "error", err)
	}
	defer pool.Close()

	rabbitCfg := rabbitmq.ConfigFromEnv()
	rabbitConn, err := rabbitmq.Connect(rabbitCfg)
	if err != nil {
		log.Fatal("rabbitmq connection failed", "error", err)
	}
	defer rabbitConn.Close()

	limit := envInt("OUTBOX_DISPATCHER_BATCH_LIMIT", 100)
	timeout := time.Duration(envInt("OUTBOX_DISPATCHER_RUN_TIMEOUT_SEC", 20)) * time.Second
	tickInterval := time.Duration(envInt("OUTBOX_DISPATCHER_TICK_INTERVAL_MS", int(defaultTickInterval/time.Millisecond))) * time.Millisecond

	outboxRepo := postgres.NewOutboxRepository(pool)
	uploadProcessPub := amqp.NewRabbitMQUploadProcessPublisher(rabbitConn)
	dispatcherSvc := service.NewDispatcherService(outboxRepo, uploadProcessPub)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	runScheduler(runCtx, log, dispatcherSvc, limit, timeout, tickInterval)

	log.Info("outbox-dispatcher started", "interval", tickInterval.String(), "batch_limit", limit, "timeout", timeout.String())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")
	cancel()
}

type dispatchRunner interface {
	DispatchPending(ctx context.Context, limit int) (service.DispatchResult, error)
}

func runScheduler(runCtx context.Context, log logger.Logger, runner dispatchRunner, limit int, timeout, tickInterval time.Duration) {
	ticker := time.NewTicker(tickInterval)
	runOnce := func() {
		runOnceCtx, cancel := context.WithTimeout(runCtx, timeout)
		defer cancel()

		res, err := runner.DispatchPending(runOnceCtx, limit)
		if err != nil && runOnceCtx.Err() == nil {
			log.Error("dispatch run failed", "error", err)
			return
		}
		log.Info("dispatch run completed", "claimed", res.Claimed, "sent", res.Sent, "retried", res.Retried, "skipped", res.Skipped)
	}

	runOnce()

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				runOnce()
			}
		}
	}()
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

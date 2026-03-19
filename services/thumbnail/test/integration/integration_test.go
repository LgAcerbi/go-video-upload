//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	testintegration "github.com/LgAcerbi/go-video-upload/pkg/testutil/integration"
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type thumbnailUploadClientFake struct{}

func (thumbnailUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{VideoID: "video-1", StoragePath: "videos/video-1/original"}, nil
}
func (thumbnailUploadClientFake) UpdateUploadStep(context.Context, string, string, string, string) (ports.StepTransitionResult, error) {
	return ports.StepTransitionResult{Applied: true}, nil
}
func (thumbnailUploadClientFake) UpdateVideoThumbnail(context.Context, string, string) error {
	return nil
}

type thumbnailFetcherFake struct{}

func (thumbnailFetcherFake) FetchToTempFile(context.Context, string, string) (string, func(), error) {
	f, err := os.CreateTemp("", "thumb-input-*.mp4")
	if err != nil {
		return "", nil, err
	}
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type thumbnailUploaderFake struct{}

func (thumbnailUploaderFake) UploadThumbnail(context.Context, string, string, io.Reader, int64) error {
	return nil
}

type thumbnailGeneratorFake struct {
	err error
}

func (g thumbnailGeneratorFake) Generate(context.Context, string) (string, func(), error) {
	if g.err != nil {
		return "", nil, g.err
	}
	f, err := os.CreateTemp("", "thumb-output-*.jpg")
	if err != nil {
		return "", nil, err
	}
	_, _ = f.Write([]byte("jpg"))
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type thumbnailStepPublisherFake struct {
	mu      sync.Mutex
	updates []string
}

func (f *thumbnailStepPublisherFake) PublishStepResult(_ context.Context, _ string, step, status, _ string) error {
	f.mu.Lock()
	f.updates = append(f.updates, step+":"+status)
	f.mu.Unlock()
	return nil
}

func TestGenerateThumbnailConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &thumbnailStepPublisherFake{}
	svc := service.NewThumbnailService(
		thumbnailUploadClientFake{},
		thumbnailFetcherFake{},
		thumbnailUploaderFake{},
		thumbnailGeneratorFake{},
		stepPub,
		"test-bucket",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "thumbnail-it"})
	go func() { _ = amqpadapter.RunGenerateThumbnailConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "generate_thumbnail", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		return len(stepPub.updates) > 0
	}, "expected done step result publish")
}

func TestGenerateThumbnailConsumer_RetryExhaustedReportsFailed_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &thumbnailStepPublisherFake{}
	svc := service.NewThumbnailService(
		thumbnailUploadClientFake{},
		thumbnailFetcherFake{},
		thumbnailUploaderFake{},
		thumbnailGeneratorFake{err: errors.New("thumbnail failed")},
		stepPub,
		"test-bucket",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "thumbnail-it"})
	go func() { _ = amqpadapter.RunGenerateThumbnailConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	err = ch.PublishWithContext(context.Background(), "", "thumbnail-generate_thumbnail", false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Headers:      amqp.Table{"x-retry-count": int32(3)},
		Body:         body,
	})
	if err != nil {
		t.Fatalf("publish with retry header: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		for _, update := range stepPub.updates {
			if update == "generate_thumbnail:failed" {
				return true
			}
		}
		return false
	}, "expected failed step result publish")
}

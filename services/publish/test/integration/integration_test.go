//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	testintegration "github.com/LgAcerbi/go-video-upload/pkg/testutil/integration"
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/publish/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/publish/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/publish/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type publishUploadClientFake struct {
	ready []ports.ReadyRendition
}

func (f *publishUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{VideoID: "video-1"}, nil
}
func (f *publishUploadClientFake) UpdateUploadStep(context.Context, string, string, string, string) (ports.StepTransitionResult, error) {
	return ports.StepTransitionResult{Applied: true}, nil
}
func (f *publishUploadClientFake) UpdateUploadStatus(context.Context, string, string) error {
	return nil
}
func (f *publishUploadClientFake) ListReadyRenditions(context.Context, string) ([]ports.ReadyRendition, error) {
	return f.ready, nil
}
func (f *publishUploadClientFake) UpdateVideoPlayback(context.Context, string, string) error {
	return nil
}

type masterUploaderFake struct{}

func (masterUploaderFake) UploadMasterPlaylist(context.Context, string, string, []byte) error {
	return nil
}

type publishStepPublisherFake struct {
	mu      sync.Mutex
	updates []string
}

func (f *publishStepPublisherFake) PublishStepResult(_ context.Context, _ string, step, status, _ string) error {
	f.mu.Lock()
	f.updates = append(f.updates, step+":"+status)
	f.mu.Unlock()
	return nil
}

func TestPublishConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &publishStepPublisherFake{}
	svc := service.NewPublishService(
		&publishUploadClientFake{ready: []ports.ReadyRendition{{Resolution: "720p", StoragePath: "videos/video-1/hls/720p/playlist.m3u8"}}},
		masterUploaderFake{},
		stepPub,
		"test-bucket",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "publish-it"})
	go func() { _ = amqpadapter.RunPublishConsumer(ctx, h.Connection, svc, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "publish", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		return len(stepPub.updates) > 0
	}, "expected done step result publish")
}

func TestPublishConsumer_RetryExhaustedReportsFailed_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &publishStepPublisherFake{}
	svc := service.NewPublishService(
		&publishUploadClientFake{ready: nil},
		masterUploaderFake{},
		stepPub,
		"test-bucket",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "publish-it"})
	go func() { _ = amqpadapter.RunPublishConsumer(ctx, h.Connection, svc, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	err = ch.PublishWithContext(context.Background(), "", "publish-publish", false, false, amqp.Publishing{
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
			if update == "publish:failed" {
				return true
			}
		}
		return false
	}, "expected failed step result publish")
}

func TestPublishConsumer_MalformedMessageToDLQ_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &publishStepPublisherFake{}
	svc := service.NewPublishService(&publishUploadClientFake{}, masterUploaderFake{}, stepPub, "test-bucket")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "publish-it"})
	go func() { _ = amqpadapter.RunPublishConsumer(ctx, h.Connection, svc, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "publish", []byte("{bad")); err != nil {
		t.Fatalf("publish bad: %v", err)
	}

	deliveries, err := ch.Consume("publish-publish.dlq", "assert", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume dlq: %v", err)
	}
	select {
	case msg := <-deliveries:
		if string(msg.Body) != "{bad" {
			t.Fatalf("unexpected dlq payload: %q", string(msg.Body))
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for dlq message")
	}
}

var _ = errors.New

//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	testintegration "github.com/LgAcerbi/go-video-upload/pkg/testutil/integration"
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/segment/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/segment/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/segment/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type segmentUploadClientFake struct {
	ready []ports.ReadyRendition
}

func (f *segmentUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{VideoID: "video-1", StoragePath: "videos/video-1/original"}, nil
}
func (f *segmentUploadClientFake) UpdateUploadStep(context.Context, string, string, string, string) (ports.StepTransitionResult, error) {
	return ports.StepTransitionResult{Applied: true}, nil
}
func (f *segmentUploadClientFake) ListReadyRenditions(context.Context, string) ([]ports.ReadyRendition, error) {
	return f.ready, nil
}

type segmentFetcherFake struct{}

func (segmentFetcherFake) FetchToTempFile(context.Context, string, string) (string, func(), error) {
	f, err := os.CreateTemp("", "segment-input-*.mp4")
	if err != nil {
		return "", nil, err
	}
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type hlsProducerFake struct {
	mu     sync.Mutex
	err    error
	called int
}

func (f *hlsProducerFake) ProduceAndUpload(context.Context, string, string, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called++
	return f.err
}

type segmentStepPublisherFake struct {
	mu      sync.Mutex
	updates []string
}

func (f *segmentStepPublisherFake) PublishStepResult(_ context.Context, _ string, step, status, _ string) error {
	f.mu.Lock()
	f.updates = append(f.updates, step+":"+status)
	f.mu.Unlock()
	return nil
}

func TestSegmentConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &segmentStepPublisherFake{}
	producer := &hlsProducerFake{}
	svc := service.NewSegmentService(
		&segmentUploadClientFake{ready: []ports.ReadyRendition{{Resolution: "720p", StoragePath: "videos/video-1/720p.mp4"}}},
		segmentFetcherFake{},
		producer,
		stepPub,
		"test-bucket",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "segment-it"})
	go func() { _ = amqpadapter.RunSegmentConsumer(ctx, h.Connection, svc, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "segment", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		return len(stepPub.updates) > 0
	}, "expected done step result publish")
}

func TestSegmentConsumer_RetryExhaustedReportsFailed_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &segmentStepPublisherFake{}
	svc := service.NewSegmentService(
		&segmentUploadClientFake{ready: []ports.ReadyRendition{{Resolution: "720p", StoragePath: "videos/video-1/720p.mp4"}}},
		segmentFetcherFake{},
		&hlsProducerFake{err: errors.New("segment failed")},
		stepPub,
		"test-bucket",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "segment-it"})
	go func() { _ = amqpadapter.RunSegmentConsumer(ctx, h.Connection, svc, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	err = ch.PublishWithContext(context.Background(), "", "segment-segment", false, false, amqp.Publishing{
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
			if update == "segment:failed" {
				return true
			}
		}
		return false
	}, "expected failed step result publish")
}

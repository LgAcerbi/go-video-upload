//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	testintegration "github.com/LgAcerbi/go-video-upload/pkg/testutil/integration"
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/metadata/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type metadataUploadClientFake struct {
	mu               sync.Mutex
	stepUpdates      []string
	videoMetaUpdates int
	renditionCreates int
}

func (f *metadataUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{VideoID: "video-1", StoragePath: "videos/video-1/original"}, nil
}

func (f *metadataUploadClientFake) UpdateUploadStep(_ context.Context, _ string, step, status, _ string) (ports.StepTransitionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stepUpdates = append(f.stepUpdates, step+":"+status)
	return ports.StepTransitionResult{Applied: true, ToStatus: status}, nil
}

func (f *metadataUploadClientFake) UpdateVideoMetadata(context.Context, string, string, float64, string) error {
	f.mu.Lock()
	f.videoMetaUpdates++
	f.mu.Unlock()
	return nil
}

func (f *metadataUploadClientFake) CreateRenditions(context.Context, string, string, int32, int32, []int32) error {
	f.mu.Lock()
	f.renditionCreates++
	f.mu.Unlock()
	return nil
}

type metadataStepPublisherFake struct {
	mu      sync.Mutex
	updates []string
}

func (f *metadataStepPublisherFake) PublishStepResult(_ context.Context, _ string, step, status, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, step+":"+status)
	return nil
}

type metadataFileFetcherFake struct{}

func (metadataFileFetcherFake) FetchToTempFile(context.Context, string, string) (string, func(), error) {
	return "test.mp4", func() {}, nil
}

type metadataExtractorFake struct {
	err error
}

func (f metadataExtractorFake) Extract(context.Context, string) (string, float64, int32, int32, error) {
	if f.err != nil {
		return "", 0, 0, 0, f.err
	}
	return "mp4", 12.5, 1920, 1080, nil
}

func TestExtractMetadataConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	uploadClient := &metadataUploadClientFake{}
	stepPublisher := &metadataStepPublisherFake{}
	svc := service.NewMetadataService(uploadClient, stepPublisher, metadataFileFetcherFake{}, metadataExtractorFake{}, "test-bucket")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "metadata-it"})
	go func() {
		_ = amqpadapter.RunExtractMetadataConsumer(ctx, h.Connection, svc, nil, log)
	}()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-happy"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "extract_metadata", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		uploadClient.mu.Lock()
		defer uploadClient.mu.Unlock()
		stepPublisher.mu.Lock()
		defer stepPublisher.mu.Unlock()
		return len(uploadClient.stepUpdates) > 0 && len(stepPublisher.updates) > 0
	}, "expected extract metadata step to complete")
}

func TestExtractMetadataConsumer_InvalidMessageToDLQ_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	svc := service.NewMetadataService(&metadataUploadClientFake{}, &metadataStepPublisherFake{}, metadataFileFetcherFake{}, metadataExtractorFake{}, "test-bucket")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "metadata-it"})
	go func() {
		_ = amqpadapter.RunExtractMetadataConsumer(ctx, h.Connection, svc, nil, log)
	}()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "extract_metadata", []byte("{invalid")); err != nil {
		t.Fatalf("publish invalid payload: %v", err)
	}

	msg := consumeOne(t, ch, "metadata-extract_metadata.dlq")
	if string(msg.Body) != "{invalid" {
		t.Fatalf("expected original payload in dlq, got %q", string(msg.Body))
	}
}

func TestExtractMetadataConsumer_RetryExhaustedReportsFailed_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	uploadClient := &metadataUploadClientFake{}
	stepPublisher := &metadataStepPublisherFake{}
	svc := service.NewMetadataService(uploadClient, stepPublisher, metadataFileFetcherFake{}, metadataExtractorFake{err: errors.New("extract failed")}, "test-bucket")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "metadata-it"})
	go func() {
		_ = amqpadapter.RunExtractMetadataConsumer(ctx, h.Connection, svc, nil, log)
	}()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	body, _ := json.Marshal(map[string]string{"upload_id": "upload-fail"})
	err = ch.PublishWithContext(context.Background(), "", "metadata-extract_metadata", false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      amqp.Table{"x-retry-count": int32(3)},
	})
	if err != nil {
		t.Fatalf("publish with retry header: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPublisher.mu.Lock()
		defer stepPublisher.mu.Unlock()
		for _, update := range stepPublisher.updates {
			if update == "extract_metadata:failed" {
				return true
			}
		}
		return false
	}, "expected failed step result publish after retry exhaustion")

	msg := consumeOne(t, ch, "metadata-extract_metadata.dlq")
	if msg.Headers["x-final-error"] == nil {
		t.Fatalf("expected x-final-error header on dlq message")
	}
}

func consumeOne(t *testing.T, ch *amqp.Channel, queue string) amqp.Delivery {
	t.Helper()
	deliveries, err := ch.Consume(queue, "integration-assert", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume queue %s: %v", queue, err)
	}
	select {
	case d := <-deliveries:
		return d
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for message on %s", queue)
	}
	return amqp.Delivery{}
}

func (f *metadataUploadClientFake) String() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return fmt.Sprintf("updates=%v", f.stepUpdates)
}

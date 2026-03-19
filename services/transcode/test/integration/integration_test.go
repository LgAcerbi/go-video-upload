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
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/transcode/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type transcodeUploadClientFake struct {
	mu          sync.Mutex
	stepUpdates []string
	pending     []ports.PendingRendition
}

func (f *transcodeUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{VideoID: "video-1", StoragePath: "videos/video-1/original"}, nil
}
func (f *transcodeUploadClientFake) UpdateUploadStep(_ context.Context, _ string, step, status, _ string) (ports.StepTransitionResult, error) {
	f.mu.Lock()
	f.stepUpdates = append(f.stepUpdates, step+":"+status)
	f.mu.Unlock()
	return ports.StepTransitionResult{Applied: true, ToStatus: status}, nil
}
func (f *transcodeUploadClientFake) ListPendingRenditions(context.Context, string) ([]ports.PendingRendition, error) {
	return f.pending, nil
}
func (f *transcodeUploadClientFake) UpdateRendition(context.Context, string, string, string, *int32, *int32, *int32, string) error {
	return nil
}

type transcodeFetcherFake struct{}

func (transcodeFetcherFake) FetchToTempFile(context.Context, string, string) (string, func(), error) {
	f, err := os.CreateTemp("", "transcode-input-*.mp4")
	if err != nil {
		return "", nil, err
	}
	_, _ = f.Write([]byte("input"))
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type transcodeUploaderFake struct{}

func (transcodeUploaderFake) UploadRendition(context.Context, string, string, io.Reader, int64) error {
	return nil
}

type transcoderFake struct {
	err error
}

func (t transcoderFake) Transcode(context.Context, string, int) (string, func(), error) {
	if t.err != nil {
		return "", nil, t.err
	}
	f, err := os.CreateTemp("", "transcode-output-*.mp4")
	if err != nil {
		return "", nil, err
	}
	_, _ = f.Write([]byte("output"))
	_ = f.Close()
	return f.Name(), func() { _ = os.Remove(f.Name()) }, nil
}

type proberFake struct{}

func (proberFake) Probe(context.Context, string) (int, int, error) { return 1280, 720, nil }

type stepResultPublisherFake struct {
	mu      sync.Mutex
	updates []string
}

func (f *stepResultPublisherFake) PublishStepResult(_ context.Context, _ string, step, status, _ string) error {
	f.mu.Lock()
	f.updates = append(f.updates, step+":"+status)
	f.mu.Unlock()
	return nil
}

func TestTranscodeConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	uploadClient := &transcodeUploadClientFake{
		pending: []ports.PendingRendition{{Resolution: "720p", Height: 720}},
	}
	stepPub := &stepResultPublisherFake{}
	svc := service.NewTranscodeService(uploadClient, transcodeFetcherFake{}, transcodeUploaderFake{}, transcoderFake{}, proberFake{}, stepPub, "test-bucket")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "transcode-it"})
	go func() { _ = amqpadapter.RunTranscodeConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "pipeline-steps", "transcode", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		return len(stepPub.updates) > 0
	}, "expected done step result publish")
}

func TestTranscodeConsumer_RetryExhaustedReportsFailed_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	stepPub := &stepResultPublisherFake{}
	svc := service.NewTranscodeService(
		&transcodeUploadClientFake{pending: []ports.PendingRendition{{Resolution: "720p", Height: 720}}},
		transcodeFetcherFake{},
		transcodeUploaderFake{},
		transcoderFake{err: errors.New("transcode failed")},
		proberFake{},
		stepPub,
		"test-bucket",
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "transcode-it"})
	go func() { _ = amqpadapter.RunTranscodeConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	err = ch.PublishWithContext(context.Background(), "", "transcode-transcode", false, false, amqp.Publishing{
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
			if update == "transcode:failed" {
				return true
			}
		}
		return false
	}, "expected failed step result publish")
}

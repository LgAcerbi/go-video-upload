//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	testintegration "github.com/LgAcerbi/go-video-upload/pkg/testutil/integration"
	amqpadapter "github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/adapters/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
	service "github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/services"
	amqp "github.com/rabbitmq/amqp091-go"
)

type orchestratorUploadClientFake struct {
	mu            sync.Mutex
	createCalls   int
	stepUpdates   []string
	statusUpdates []string
}

func (f *orchestratorUploadClientFake) GetUploadProcessingContext(context.Context, string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{}, nil
}
func (f *orchestratorUploadClientFake) CreateUploadSteps(context.Context, string, []string) error {
	f.mu.Lock()
	f.createCalls++
	f.mu.Unlock()
	return nil
}
func (f *orchestratorUploadClientFake) UpdateUploadStep(_ context.Context, _ string, step, status, _ string) (ports.StepTransitionResult, error) {
	f.mu.Lock()
	f.stepUpdates = append(f.stepUpdates, step+":"+status)
	f.mu.Unlock()
	return ports.StepTransitionResult{Applied: true, ToStatus: status}, nil
}
func (f *orchestratorUploadClientFake) UpdateUploadStatus(_ context.Context, _ string, status string) error {
	f.mu.Lock()
	f.statusUpdates = append(f.statusUpdates, status)
	f.mu.Unlock()
	return nil
}

type orchestratorStepPublisherFake struct {
	mu    sync.Mutex
	steps []string
}

func (f *orchestratorStepPublisherFake) PublishStep(_ context.Context, step, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.steps = append(f.steps, step)
	return nil
}

func TestUploadProcessConsumer_HappyPath_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	uploadClient := &orchestratorUploadClientFake{}
	stepPub := &orchestratorStepPublisherFake{}
	svc := service.NewOrchestratorService(uploadClient, stepPub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "orchestrator-it"})
	go func() { _ = amqpadapter.RunUploadProcessConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()
	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "", "upload-process", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		return len(stepPub.steps) >= 2
	}, "expected initial step publishes")
}

func TestStepResultConsumer_Progression_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	uploadClient := &orchestratorUploadClientFake{}
	stepPub := &orchestratorStepPublisherFake{}
	svc := service.NewOrchestratorService(uploadClient, stepPub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "orchestrator-it"})
	go func() { _ = amqpadapter.RunStepResultConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	body, _ := json.Marshal(map[string]string{"upload_id": "upload-1", "step": "extract_metadata", "status": "done"})
	if err := pkgrabbitmq.Publish(context.Background(), ch, "", "upload-process-step", body); err != nil {
		t.Fatalf("publish: %v", err)
	}

	testintegration.Eventually(t, 8*time.Second, 100*time.Millisecond, func() bool {
		stepPub.mu.Lock()
		defer stepPub.mu.Unlock()
		for _, s := range stepPub.steps {
			if s == "transcode" {
				return true
			}
		}
		return false
	}, "expected transcode to be triggered after extract_metadata done")
}

func TestUploadProcessConsumer_InvalidMessageToDLQ_Integration(t *testing.T) {
	h := testintegration.StartRabbitHarness(t)
	defer h.Close(t)

	svc := service.NewOrchestratorService(&orchestratorUploadClientFake{}, &orchestratorStepPublisherFake{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := logger.New(&logger.Config{Service: "orchestrator-it"})
	go func() { _ = amqpadapter.RunUploadProcessConsumer(ctx, h.Connection, svc, nil, log) }()

	ch, err := h.Connection.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	if err := pkgrabbitmq.Publish(context.Background(), ch, "", "upload-process", []byte("{bad-json")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	msg := consumeOne(t, ch, "upload-process.dlq")
	if string(msg.Body) != "{bad-json" {
		t.Fatalf("expected invalid payload in dlq, got %q", string(msg.Body))
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

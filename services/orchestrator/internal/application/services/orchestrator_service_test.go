package service

import (
	"context"
	"testing"

	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
)

type fakeUploadClient struct {
	createStepsCalls  int
	statusUpdates     []string
	stepUpdates       map[string][]string
	stepResultsByKey  map[string]ports.StepTransitionResult
}

func newFakeUploadClient() *fakeUploadClient {
	return &fakeUploadClient{
		stepUpdates:      make(map[string][]string),
		stepResultsByKey: make(map[string]ports.StepTransitionResult),
	}
}

func key(step, status string) string {
	return step + "|" + status
}

func (f *fakeUploadClient) GetUploadProcessingContext(ctx context.Context, uploadID string) (*ports.UploadProcessingContext, error) {
	return &ports.UploadProcessingContext{}, nil
}

func (f *fakeUploadClient) CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error {
	f.createStepsCalls++
	return nil
}

func (f *fakeUploadClient) UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) (ports.StepTransitionResult, error) {
	f.stepUpdates[step] = append(f.stepUpdates[step], status)
	if res, ok := f.stepResultsByKey[key(step, status)]; ok {
		return res, nil
	}
	return ports.StepTransitionResult{Applied: true, ToStatus: status}, nil
}

func (f *fakeUploadClient) UpdateUploadStatus(ctx context.Context, uploadID, status string) error {
	f.statusUpdates = append(f.statusUpdates, status)
	return nil
}

type fakeStepPublisher struct {
	publishedSteps []string
}

func (f *fakeStepPublisher) PublishStep(ctx context.Context, step, uploadID string) error {
	f.publishedSteps = append(f.publishedSteps, step)
	return nil
}

func TestHandleStepResult_DuplicateDoneDoesNotTriggerDownstreamTwice(t *testing.T) {
	ctx := context.Background()
	uploadClient := newFakeUploadClient()
	stepPublisher := &fakeStepPublisher{}
	svc := NewOrchestratorService(uploadClient, stepPublisher)

	if err := svc.HandleStepResult(ctx, "upload-1", "extract_metadata", statusDone, ""); err != nil {
		t.Fatalf("first done: %v", err)
	}
	uploadClient.stepResultsByKey[key("extract_metadata", statusDone)] = ports.StepTransitionResult{Applied: false, FromStatus: "done", ToStatus: statusDone, FailureReason: "invalid_transition"}
	if err := svc.HandleStepResult(ctx, "upload-1", "extract_metadata", statusDone, ""); err != nil {
		t.Fatalf("duplicate done: %v", err)
	}

	if len(stepPublisher.publishedSteps) != 1 || stepPublisher.publishedSteps[0] != "transcode" {
		t.Fatalf("expected exactly one downstream publish to transcode, got %+v", stepPublisher.publishedSteps)
	}
}

func TestHandleStepResult_OutOfOrderDoneDoesNotAdvance(t *testing.T) {
	ctx := context.Background()
	uploadClient := newFakeUploadClient()
	uploadClient.stepResultsByKey[key("transcode", statusDone)] = ports.StepTransitionResult{Applied: false, FromStatus: "pending", ToStatus: statusDone, FailureReason: "invalid_transition"}
	stepPublisher := &fakeStepPublisher{}
	svc := NewOrchestratorService(uploadClient, stepPublisher)

	if err := svc.HandleStepResult(ctx, "upload-1", "transcode", statusDone, ""); err != nil {
		t.Fatalf("out-of-order transcode done: %v", err)
	}
	if len(stepPublisher.publishedSteps) != 0 {
		t.Fatalf("expected no downstream publish, got %+v", stepPublisher.publishedSteps)
	}
}

func TestHandleStepResult_ValidPipelineProgression(t *testing.T) {
	ctx := context.Background()
	uploadClient := newFakeUploadClient()
	stepPublisher := &fakeStepPublisher{}
	svc := NewOrchestratorService(uploadClient, stepPublisher)

	if err := svc.ProcessUploadProcess(ctx, "upload-1"); err != nil {
		t.Fatalf("ProcessUploadProcess: %v", err)
	}
	if len(stepPublisher.publishedSteps) != 2 {
		t.Fatalf("expected two initial steps, got %+v", stepPublisher.publishedSteps)
	}

	stepPublisher.publishedSteps = nil
	if err := svc.HandleStepResult(ctx, "upload-1", "extract_metadata", statusDone, ""); err != nil {
		t.Fatalf("extract_metadata done: %v", err)
	}
	if err := svc.HandleStepResult(ctx, "upload-1", "transcode", statusDone, ""); err != nil {
		t.Fatalf("transcode done: %v", err)
	}
	if err := svc.HandleStepResult(ctx, "upload-1", "segment", statusDone, ""); err != nil {
		t.Fatalf("segment done: %v", err)
	}
	if err := svc.HandleStepResult(ctx, "upload-1", "publish", statusDone, ""); err != nil {
		t.Fatalf("publish done: %v", err)
	}

	if len(stepPublisher.publishedSteps) != 3 {
		t.Fatalf("expected transcode, segment, publish triggers, got %+v", stepPublisher.publishedSteps)
	}
	if len(uploadClient.statusUpdates) != 1 || uploadClient.statusUpdates[0] != "finished" {
		t.Fatalf("expected upload status finished once, got %+v", uploadClient.statusUpdates)
	}
}

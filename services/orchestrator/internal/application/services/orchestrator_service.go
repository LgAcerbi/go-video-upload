package service

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
)

const (
	statusDone       = "done"
	statusProcessing = "processing"
	statusCanceled   = "canceled"
)

var pipelineSteps = []string{
	"extract_metadata",
	"transcode",
	"generate_thumbnail",
	"segment",
	"publish",
}

const firstStep = "extract_metadata"

type OrchestratorService struct {
	uploadClient   ports.UploadStateClient
	stepPublisher  ports.StepPublisher
}

func NewOrchestratorService(uploadClient ports.UploadStateClient, stepPublisher ports.StepPublisher) *OrchestratorService {
	return &OrchestratorService{
		uploadClient:  uploadClient,
		stepPublisher: stepPublisher,
	}
}

func (s *OrchestratorService) ProcessUploadProcess(ctx context.Context, uploadID string) error {
	if err := s.uploadClient.CreateUploadSteps(ctx, uploadID, pipelineSteps); err != nil {
		return err
	}
	return s.triggerStep(ctx, firstStep, uploadID)
}

func (s *OrchestratorService) HandleStepResult(ctx context.Context, uploadID, step, status, errorMessage string) error {
	if status == models.UploadStatusFailed {
		if err := s.uploadClient.UpdateUploadStatus(ctx, uploadID, models.UploadStatusFailed); err != nil {
			return err
		}
		for _, st := range pipelineSteps {
			_ = s.uploadClient.UpdateUploadStep(ctx, uploadID, st, statusCanceled, errorMessage)
		}
		return nil
	}
	if status != statusDone {
		return nil
	}
	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, step, statusDone, ""); err != nil {
		return err
	}
	switch step {
	case "extract_metadata":
		if err := s.triggerStep(ctx, "transcode", uploadID); err != nil {
			return err
		}
		return s.triggerStep(ctx, "generate_thumbnail", uploadID)
	case "transcode":
		return s.triggerStep(ctx, "segment", uploadID)
	case "generate_thumbnail":
		return nil
	case "segment":
		return s.triggerStep(ctx, "publish", uploadID)
	case "publish":
		return s.uploadClient.UpdateUploadStatus(ctx, uploadID, models.UploadStatusFinished)
	default:
		return nil
	}
}

func (s *OrchestratorService) triggerStep(ctx context.Context, step, uploadID string) error {
	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, step, statusProcessing, ""); err != nil {
		return err
	}
	return s.stepPublisher.PublishStep(ctx, step, uploadID)
}

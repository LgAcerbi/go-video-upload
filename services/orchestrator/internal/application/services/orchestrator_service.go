package service

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
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
	uploadClient ports.UploadStateClient
	stepPublisher ports.StepPublisher
}

func NewOrchestratorService(uploadClient ports.UploadStateClient, stepPublisher ports.StepPublisher) *OrchestratorService {
	return &OrchestratorService{
		uploadClient:  uploadClient,
		stepPublisher: stepPublisher,
	}
}

func (s *OrchestratorService) ProcessUploadProcess(ctx context.Context, videoID, uploadID, storagePath string) error {
	if err := s.uploadClient.CreateUploadSteps(ctx, uploadID, pipelineSteps); err != nil {
		return err
	}
	return s.stepPublisher.PublishStep(ctx, firstStep, videoID, uploadID, storagePath)
}

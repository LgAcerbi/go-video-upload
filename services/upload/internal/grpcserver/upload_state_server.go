package grpcserver

import (
	"context"
	"time"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
)

var _ upload.UploadStateServiceServer = (*UploadStateServer)(nil)

type UploadStateServer struct {
	upload.UnimplementedUploadStateServiceServer
	uploadRepo     ports.UploadRepository
	uploadStepRepo ports.UploadStepRepository
	videoRepo      ports.VideoRepository
}

func NewUploadStateServer(uploadRepo ports.UploadRepository, uploadStepRepo ports.UploadStepRepository, videoRepo ports.VideoRepository) *UploadStateServer {
	return &UploadStateServer{
		uploadRepo:     uploadRepo,
		uploadStepRepo: uploadStepRepo,
		videoRepo:      videoRepo,
	}
}

func (s *UploadStateServer) UpdateUploadStatus(ctx context.Context, req *upload.UpdateUploadStatusRequest) (*upload.UpdateUploadStatusResponse, error) {
	if req == nil || req.UploadId == "" || req.Status == "" {
		return &upload.UpdateUploadStatusResponse{}, nil
	}
	if err := s.uploadRepo.UpdateStatus(ctx, req.UploadId, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStatusResponse{}, nil
}

func (s *UploadStateServer) UpdateUploadStep(ctx context.Context, req *upload.UpdateUploadStepRequest) (*upload.UpdateUploadStepResponse, error) {
	if req == nil || req.UploadId == "" || req.Step == "" || req.Status == "" {
		return &upload.UpdateUploadStepResponse{}, nil
	}
	if err := s.uploadStepRepo.UpdateStepStatus(ctx, req.UploadId, req.Step, req.Status, req.ErrorMessage); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStepResponse{}, nil
}

func (s *UploadStateServer) UpdateVideoMetadata(ctx context.Context, req *upload.UpdateVideoMetadataRequest) (*upload.UpdateVideoMetadataResponse, error) {
	if req == nil || req.VideoId == "" {
		return &upload.UpdateVideoMetadataResponse{}, nil
	}
	v, err := s.videoRepo.GetByID(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}
	if req.Format != "" {
		v.Format = req.Format
	}
	if req.DurationSec > 0 {
		v.DurationSec = &req.DurationSec
	}
	if req.Status != "" {
		v.Status = req.Status
	}
	v.UpdatedAt = time.Now()
	if err := s.videoRepo.Update(ctx, v); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoMetadataResponse{}, nil
}

func (s *UploadStateServer) CreateUploadSteps(ctx context.Context, req *upload.CreateUploadStepsRequest) (*upload.CreateUploadStepsResponse, error) {
	if req == nil || req.UploadId == "" || len(req.Steps) == 0 {
		return &upload.CreateUploadStepsResponse{}, nil
	}
	if err := s.uploadStepRepo.CreateSteps(ctx, req.UploadId, req.Steps); err != nil {
		return nil, err
	}
	return &upload.CreateUploadStepsResponse{}, nil
}

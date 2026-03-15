package grpcserver

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/services"
)

var _ upload.UploadStateServiceServer = (*UploadStateServer)(nil)

// UploadStateServer is the gRPC driver for UploadStateService. It delegates all use-cases to the service.
type UploadStateServer struct {
	upload.UnimplementedUploadStateServiceServer
	svc *service.UploadService
}

// NewUploadStateServer returns a gRPC server that delegates to the upload service.
func NewUploadStateServer(svc *service.UploadService) *UploadStateServer {
	return &UploadStateServer{svc: svc}
}

func (s *UploadStateServer) UpdateUploadStatus(ctx context.Context, req *upload.UpdateUploadStatusRequest) (*upload.UpdateUploadStatusResponse, error) {
	if req == nil || req.UploadId == "" || req.Status == "" {
		return &upload.UpdateUploadStatusResponse{}, nil
	}
	if err := s.svc.UpdateUploadStatus(ctx, req.UploadId, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStatusResponse{}, nil
}

func (s *UploadStateServer) UpdateUploadStep(ctx context.Context, req *upload.UpdateUploadStepRequest) (*upload.UpdateUploadStepResponse, error) {
	if req == nil || req.UploadId == "" || req.Step == "" || req.Status == "" {
		return &upload.UpdateUploadStepResponse{}, nil
	}
	if err := s.svc.UpdateUploadStep(ctx, req.UploadId, req.Step, req.Status, req.ErrorMessage); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStepResponse{}, nil
}

func (s *UploadStateServer) UpdateVideoMetadata(ctx context.Context, req *upload.UpdateVideoMetadataRequest) (*upload.UpdateVideoMetadataResponse, error) {
	if req == nil || req.VideoId == "" {
		return &upload.UpdateVideoMetadataResponse{}, nil
	}
	if err := s.svc.UpdateVideoMetadata(ctx, req.VideoId, req.Format, req.DurationSec, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoMetadataResponse{}, nil
}

func (s *UploadStateServer) CreateUploadSteps(ctx context.Context, req *upload.CreateUploadStepsRequest) (*upload.CreateUploadStepsResponse, error) {
	if req == nil || req.UploadId == "" || len(req.Steps) == 0 {
		return &upload.CreateUploadStepsResponse{}, nil
	}
	if err := s.svc.CreateUploadSteps(ctx, req.UploadId, req.Steps); err != nil {
		return nil, err
	}
	return &upload.CreateUploadStepsResponse{}, nil
}

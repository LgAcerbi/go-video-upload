package grpcserver

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	service "github.com/LgAcerbi/go-video-upload/services/upload/internal/application/services"
)

var _ upload.UploadStateServiceServer = (*UploadStateController)(nil)

type UploadStateController struct {
	upload.UnimplementedUploadStateServiceServer
	svc *service.UploadService
}

func NewUploadStateController(svc *service.UploadService) *UploadStateController {
	return &UploadStateController{svc: svc}
}

func (c *UploadStateController) UpdateUploadStatus(ctx context.Context, req *upload.UpdateUploadStatusRequest) (*upload.UpdateUploadStatusResponse, error) {
	if req == nil || req.UploadId == "" || req.Status == "" {
		return &upload.UpdateUploadStatusResponse{}, nil
	}
	if err := c.svc.UpdateUploadStatus(ctx, req.UploadId, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStatusResponse{}, nil
}

func (c *UploadStateController) UpdateUploadStep(ctx context.Context, req *upload.UpdateUploadStepRequest) (*upload.UpdateUploadStepResponse, error) {
	if req == nil || req.UploadId == "" || req.Step == "" || req.Status == "" {
		return &upload.UpdateUploadStepResponse{}, nil
	}
	if err := c.svc.UpdateUploadStep(ctx, req.UploadId, req.Step, req.Status, req.ErrorMessage); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStepResponse{}, nil
}

func (c *UploadStateController) UpdateVideoMetadata(ctx context.Context, req *upload.UpdateVideoMetadataRequest) (*upload.UpdateVideoMetadataResponse, error) {
	if req == nil || req.VideoId == "" {
		return &upload.UpdateVideoMetadataResponse{}, nil
	}
	if err := c.svc.UpdateVideoMetadata(ctx, req.VideoId, req.Format, req.DurationSec, req.Status, req.Width, req.Height); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoMetadataResponse{}, nil
}

func (c *UploadStateController) CreateUploadSteps(ctx context.Context, req *upload.CreateUploadStepsRequest) (*upload.CreateUploadStepsResponse, error) {
	if req == nil || req.UploadId == "" || len(req.Steps) == 0 {
		return &upload.CreateUploadStepsResponse{}, nil
	}
	if err := c.svc.CreateUploadSteps(ctx, req.UploadId, req.Steps); err != nil {
		return nil, err
	}
	return &upload.CreateUploadStepsResponse{}, nil
}

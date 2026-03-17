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
	if err := c.svc.UpdateVideoMetadata(ctx, req.VideoId, req.Format, req.DurationSec, req.Status); err != nil {
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

func (c *UploadStateController) CreateRenditions(ctx context.Context, req *upload.CreateRenditionsRequest) (*upload.CreateRenditionsResponse, error) {
	if req == nil || req.VideoId == "" {
		return &upload.CreateRenditionsResponse{}, nil
	}
	if err := c.svc.CreateRenditions(ctx, req.VideoId, req.OriginalStoragePath, req.OriginalWidth, req.OriginalHeight, req.TargetHeights); err != nil {
		return nil, err
	}
	return &upload.CreateRenditionsResponse{}, nil
}

func (c *UploadStateController) ListPendingRenditions(ctx context.Context, req *upload.ListPendingRenditionsRequest) (*upload.ListPendingRenditionsResponse, error) {
	if req == nil || req.VideoId == "" {
		return &upload.ListPendingRenditionsResponse{}, nil
	}
	renditions, err := c.svc.ListPendingRenditions(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}
	out := make([]*upload.PendingRendition, len(renditions))
	for i, r := range renditions {
		h := int32(0)
		if r.Height != nil {
			h = int32(*r.Height)
		}
		out[i] = &upload.PendingRendition{Resolution: r.Resolution, Height: h}
	}
	return &upload.ListPendingRenditionsResponse{Renditions: out}, nil
}

func (c *UploadStateController) UpdateRendition(ctx context.Context, req *upload.UpdateRenditionRequest) (*upload.UpdateRenditionResponse, error) {
	if req == nil || req.VideoId == "" || req.Resolution == "" || req.StoragePath == "" {
		return &upload.UpdateRenditionResponse{}, nil
	}
	var width, height, bitrate *int32
	if req.Width > 0 {
		width = &req.Width
	}
	if req.Height > 0 {
		height = &req.Height
	}
	if req.BitrateKbps > 0 {
		bitrate = &req.BitrateKbps
	}
	if err := c.svc.UpdateRendition(ctx, req.VideoId, req.Resolution, req.StoragePath, width, height, bitrate, req.Format); err != nil {
		return nil, err
	}
	return &upload.UpdateRenditionResponse{}, nil
}

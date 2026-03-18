package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func (c *UploadStateController) GetUploadProcessingContext(ctx context.Context, req *upload.GetUploadProcessingContextRequest) (*upload.GetUploadProcessingContextResponse, error) {
	if req == nil || req.UploadId == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required")
	}
	u, err := c.svc.GetUploadByID(ctx, req.UploadId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "upload not found: %v", err)
	}
	resp := &upload.GetUploadProcessingContextResponse{
		UploadId:         u.ID,
		VideoId:          u.VideoID,
		StoragePath:      u.StoragePath,
		Status:           u.Status,
		CreatedAtUnixSec: u.CreatedAt.Unix(),
		UpdatedAtUnixSec: u.UpdatedAt.Unix(),
	}
	if u.ExpiresAt != nil {
		resp.ExpiresAtUnixSec = u.ExpiresAt.Unix()
	}
	return resp, nil
}

func (c *UploadStateController) UpdateUploadStatus(ctx context.Context, req *upload.UpdateUploadStatusRequest) (*upload.UpdateUploadStatusResponse, error) {
	if req == nil || req.UploadId == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required")
	}
	if req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}
	if err := c.svc.UpdateUploadStatus(ctx, req.UploadId, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStatusResponse{}, nil
}

func (c *UploadStateController) UpdateUploadStep(ctx context.Context, req *upload.UpdateUploadStepRequest) (*upload.UpdateUploadStepResponse, error) {
	if req == nil || req.UploadId == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required")
	}
	if req.Step == "" {
		return nil, status.Error(codes.InvalidArgument, "step is required")
	}
	if req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}
	if err := c.svc.UpdateUploadStep(ctx, req.UploadId, req.Step, req.Status, req.ErrorMessage); err != nil {
		return nil, err
	}
	return &upload.UpdateUploadStepResponse{}, nil
}

func (c *UploadStateController) UpdateVideoMetadata(ctx context.Context, req *upload.UpdateVideoMetadataRequest) (*upload.UpdateVideoMetadataResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	if err := c.svc.UpdateVideoMetadata(ctx, req.VideoId, req.Format, req.DurationSec, req.Status); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoMetadataResponse{}, nil
}

func (c *UploadStateController) UpdateVideoThumbnail(ctx context.Context, req *upload.UpdateVideoThumbnailRequest) (*upload.UpdateVideoThumbnailResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	if req.ThumbnailStoragePath == "" {
		return nil, status.Error(codes.InvalidArgument, "thumbnail_storage_path is required")
	}
	if err := c.svc.UpdateVideoThumbnail(ctx, req.VideoId, req.ThumbnailStoragePath); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoThumbnailResponse{}, nil
}

func (c *UploadStateController) CreateUploadSteps(ctx context.Context, req *upload.CreateUploadStepsRequest) (*upload.CreateUploadStepsResponse, error) {
	if req == nil || req.UploadId == "" {
		return nil, status.Error(codes.InvalidArgument, "upload_id is required")
	}
	if len(req.Steps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "steps cannot be empty")
	}
	if err := c.svc.CreateUploadSteps(ctx, req.UploadId, req.Steps); err != nil {
		return nil, err
	}
	return &upload.CreateUploadStepsResponse{}, nil
}

func (c *UploadStateController) CreateRenditions(ctx context.Context, req *upload.CreateRenditionsRequest) (*upload.CreateRenditionsResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	if err := c.svc.CreateRenditions(ctx, req.VideoId, req.OriginalStoragePath, req.OriginalWidth, req.OriginalHeight, req.TargetHeights); err != nil {
		return nil, err
	}
	return &upload.CreateRenditionsResponse{}, nil
}

func (c *UploadStateController) ListPendingRenditions(ctx context.Context, req *upload.ListPendingRenditionsRequest) (*upload.ListPendingRenditionsResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
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

func (c *UploadStateController) ListReadyRenditions(ctx context.Context, req *upload.ListReadyRenditionsRequest) (*upload.ListReadyRenditionsResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	renditions, err := c.svc.ListReadyRenditions(ctx, req.VideoId)
	if err != nil {
		return nil, err
	}
	out := make([]*upload.ReadyRendition, len(renditions))
	for i, r := range renditions {
		path := ""
		if r.StoragePath != nil {
			path = *r.StoragePath
		}
		out[i] = &upload.ReadyRendition{Resolution: r.Resolution, StoragePath: path}
	}
	return &upload.ListReadyRenditionsResponse{Renditions: out}, nil
}

func (c *UploadStateController) UpdateRendition(ctx context.Context, req *upload.UpdateRenditionRequest) (*upload.UpdateRenditionResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	if req.Resolution == "" {
		return nil, status.Error(codes.InvalidArgument, "resolution is required")
	}
	if req.StoragePath == "" {
		return nil, status.Error(codes.InvalidArgument, "storage_path is required")
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

func (c *UploadStateController) UpdateVideoPlayback(ctx context.Context, req *upload.UpdateVideoPlaybackRequest) (*upload.UpdateVideoPlaybackResponse, error) {
	if req == nil || req.VideoId == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id is required")
	}
	if req.HlsMasterPath == "" {
		return nil, status.Error(codes.InvalidArgument, "hls_master_path is required")
	}
	if err := c.svc.UpdateVideoPlayback(ctx, req.VideoId, req.HlsMasterPath); err != nil {
		return nil, err
	}
	return &upload.UpdateVideoPlaybackResponse{}, nil
}

func (c *UploadStateController) ExpireStaleUploads(ctx context.Context, req *upload.ExpireStaleUploadsRequest) (*upload.ExpireStaleUploadsResponse, error) {
	limit := 0
	if req != nil && req.Limit > 0 {
		limit = int(req.Limit)
	}
	res, err := c.svc.ExpireStaleUploads(ctx, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "expire stale uploads: %v", err)
	}
	return &upload.ExpireStaleUploadsResponse{
		Found:   int32(res.Found),
		Expired: int32(res.Expired),
		Skipped: int32(res.Skipped),
	}, nil
}

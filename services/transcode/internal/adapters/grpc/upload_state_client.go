package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

type UploadStateClient struct {
	client upload.UploadStateServiceClient
	conn   *grpc.ClientConn
}

func NewUploadStateClient(ctx context.Context, target string) (*UploadStateClient, error) {
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := upload.NewUploadStateServiceClient(conn)
	return &UploadStateClient{client: client, conn: conn}, nil
}

func (c *UploadStateClient) GetUploadProcessingContext(ctx context.Context, uploadID string) (*ports.UploadProcessingContext, error) {
	resp, err := c.client.GetUploadProcessingContext(ctx, &upload.GetUploadProcessingContextRequest{UploadId: uploadID})
	if err != nil {
		return nil, err
	}
	return &ports.UploadProcessingContext{
		VideoID:     resp.GetVideoId(),
		StoragePath: resp.GetStoragePath(),
	}, nil
}

func (c *UploadStateClient) UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) (ports.StepTransitionResult, error) {
	resp, err := c.client.UpdateUploadStep(ctx, &upload.UpdateUploadStepRequest{
		UploadId:     uploadID,
		Step:         step,
		Status:       status,
		ErrorMessage: errorMessage,
	})
	if err != nil {
		return ports.StepTransitionResult{}, err
	}
	return ports.StepTransitionResult{
		Applied:       resp.GetApplied(),
		FromStatus:    resp.GetFromStatus(),
		ToStatus:      resp.GetToStatus(),
		FailureReason: resp.GetFailureReason(),
	}, nil
}

func (c *UploadStateClient) ListPendingRenditions(ctx context.Context, videoID string) ([]ports.PendingRendition, error) {
	resp, err := c.client.ListPendingRenditions(ctx, &upload.ListPendingRenditionsRequest{VideoId: videoID})
	if err != nil {
		return nil, err
	}
	out := make([]ports.PendingRendition, len(resp.Renditions))
	for i, r := range resp.Renditions {
		out[i] = ports.PendingRendition{Resolution: r.Resolution, Height: int(r.Height)}
	}
	return out, nil
}

func (c *UploadStateClient) UpdateRendition(ctx context.Context, videoID, resolution, storagePath string, width, height, bitrateKbps *int32, format string) error {
	req := &upload.UpdateRenditionRequest{
		VideoId:     videoID,
		Resolution:  resolution,
		StoragePath: storagePath,
		Format:      format,
	}
	if width != nil {
		req.Width = *width
	}
	if height != nil {
		req.Height = *height
	}
	if bitrateKbps != nil {
		req.BitrateKbps = *bitrateKbps
	}
	_, err := c.client.UpdateRendition(ctx, req)
	return err
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}

var _ ports.UploadStateClient = (*UploadStateClient)(nil)

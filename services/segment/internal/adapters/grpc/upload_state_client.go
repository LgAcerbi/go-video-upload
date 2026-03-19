package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/segment/internal/application/ports"
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

func (c *UploadStateClient) ListReadyRenditions(ctx context.Context, videoID string) ([]ports.ReadyRendition, error) {
	resp, err := c.client.ListReadyRenditions(ctx, &upload.ListReadyRenditionsRequest{VideoId: videoID})
	if err != nil {
		return nil, err
	}
	out := make([]ports.ReadyRendition, len(resp.Renditions))
	for i, r := range resp.Renditions {
		out[i] = ports.ReadyRendition{Resolution: r.Resolution, StoragePath: r.StoragePath}
	}
	return out, nil
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}

var _ ports.UploadStateClient = (*UploadStateClient)(nil)

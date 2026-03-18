package grpcclient

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func (c *UploadStateClient) UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error {
	_, err := c.client.UpdateUploadStep(ctx, &upload.UpdateUploadStepRequest{
		UploadId:     uploadID,
		Step:         step,
		Status:       status,
		ErrorMessage: errorMessage,
	})
	return err
}

func (c *UploadStateClient) UpdateVideoThumbnail(ctx context.Context, videoID, thumbnailStoragePath string) error {
	_, err := c.client.UpdateVideoThumbnail(ctx, &upload.UpdateVideoThumbnailRequest{
		VideoId:              videoID,
		ThumbnailStoragePath: thumbnailStoragePath,
	})
	return err
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}

var _ ports.UploadStateClient = (*UploadStateClient)(nil)


package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
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

func (c *UploadStateClient) UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error {
	_, err := c.client.UpdateUploadStep(ctx, &upload.UpdateUploadStepRequest{
		UploadId:     uploadID,
		Step:         step,
		Status:       status,
		ErrorMessage: errorMessage,
	})
	return err
}

func (c *UploadStateClient) UpdateVideoMetadata(ctx context.Context, videoID, format string, durationSec float64, status string) error {
	_, err := c.client.UpdateVideoMetadata(ctx, &upload.UpdateVideoMetadataRequest{
		VideoId:     videoID,
		Format:      format,
		DurationSec: durationSec,
		Status:      status,
	})
	return err
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}

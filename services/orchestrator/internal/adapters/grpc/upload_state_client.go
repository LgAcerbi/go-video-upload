package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
)

type UploadStateClient struct {
	client upload.UploadStateServiceClient
	conn   *grpc.ClientConn
}

func NewUploadStateClient(ctx context.Context, target string) (ports.UploadStateClient, error) {
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := upload.NewUploadStateServiceClient(conn)
	return &UploadStateClient{client: client, conn: conn}, nil
}

func (c *UploadStateClient) CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error {
	_, err := c.client.CreateUploadSteps(ctx, &upload.CreateUploadStepsRequest{
		UploadId: uploadID,
		Steps:    steps,
	})
	return err
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}

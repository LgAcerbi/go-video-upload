package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/LgAcerbi/go-video-upload/proto/upload"
	"github.com/LgAcerbi/go-video-upload/services/expirer/internal/application/ports"
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
	return &UploadStateClient{
		client: upload.NewUploadStateServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *UploadStateClient) ExpireStaleUploads(ctx context.Context, limit int) (ports.ExpireResult, error) {
	req := &upload.ExpireStaleUploadsRequest{}
	if limit > 0 {
		req.Limit = int32(limit)
	}
	resp, err := c.client.ExpireStaleUploads(ctx, req)
	if err != nil {
		return ports.ExpireResult{}, err
	}
	return ports.ExpireResult{
		Found:   int(resp.GetFound()),
		Expired: int(resp.GetExpired()),
		Skipped: int(resp.GetSkipped()),
	}, nil
}

func (c *UploadStateClient) Close() error {
	return c.conn.Close()
}


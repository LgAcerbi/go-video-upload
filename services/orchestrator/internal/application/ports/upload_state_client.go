package ports

import "context"

type UploadStateClient interface {
	CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error
}

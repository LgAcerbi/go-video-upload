package ports

import "context"

type UploadStateClient interface {
	CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateUploadStatus(ctx context.Context, uploadID, status string) error
}

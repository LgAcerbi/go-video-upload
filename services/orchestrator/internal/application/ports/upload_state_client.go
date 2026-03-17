package ports

import "context"

// UploadProcessingContext is returned by GetUploadProcessingContext for pipeline step dispatch.
type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateUploadStatus(ctx context.Context, uploadID, status string) error
}

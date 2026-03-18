package ports

import "context"

// ReadyRendition is a rendition with storage path (for segment step).
type ReadyRendition struct {
	Resolution  string
	StoragePath string
}

// UploadProcessingContext is returned by GetUploadProcessingContext.
type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	ListReadyRenditions(ctx context.Context, videoID string) ([]ReadyRendition, error)
}

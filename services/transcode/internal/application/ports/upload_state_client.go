package ports

import "context"

type PendingRendition struct {
	Resolution string
	Height     int
}

// UploadProcessingContext is returned by GetUploadProcessingContext for pipeline step processing.
type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	ListPendingRenditions(ctx context.Context, videoID string) ([]PendingRendition, error)
	UpdateRendition(ctx context.Context, videoID, resolution, storagePath string, width, height, bitrateKbps *int32, format string) error
}

package ports

import "context"

// UploadProcessingContext is returned by GetUploadProcessingContext for pipeline step processing.
type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateVideoMetadata(ctx context.Context, videoID, format string, durationSec float64, status string) error
	CreateRenditions(ctx context.Context, videoID, originalStoragePath string, originalWidth, originalHeight int32, targetHeights []int32) error
}

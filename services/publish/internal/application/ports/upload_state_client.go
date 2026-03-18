package ports

import "context"

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateUploadStatus(ctx context.Context, uploadID, status string) error
	ListReadyRenditions(ctx context.Context, videoID string) ([]ReadyRendition, error)
	UpdateVideoPlayback(ctx context.Context, videoID, hlsMasterPath string) error
}

type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type ReadyRendition struct {
	Resolution  string
	StoragePath string
}

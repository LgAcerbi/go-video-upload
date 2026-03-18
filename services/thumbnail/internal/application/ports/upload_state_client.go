package ports

import "context"

type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateVideoThumbnail(ctx context.Context, videoID, thumbnailStoragePath string) error
}


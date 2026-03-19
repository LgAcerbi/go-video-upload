package ports

import "context"

type UploadProcessingContext struct {
	VideoID     string
	StoragePath string
}

type StepTransitionResult struct {
	Applied       bool
	FromStatus    string
	ToStatus      string
	FailureReason string
}

type UploadStateClient interface {
	GetUploadProcessingContext(ctx context.Context, uploadID string) (*UploadProcessingContext, error)
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) (StepTransitionResult, error)
	UpdateVideoThumbnail(ctx context.Context, videoID, thumbnailStoragePath string) error
}


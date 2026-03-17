package ports

import "context"

type PendingRendition struct {
	Resolution string
	Height     int
}

type UploadStateClient interface {
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	ListPendingRenditions(ctx context.Context, videoID string) ([]PendingRendition, error)
	UpdateRendition(ctx context.Context, videoID, resolution, storagePath string, width, height, bitrateKbps *int32, format string) error
}

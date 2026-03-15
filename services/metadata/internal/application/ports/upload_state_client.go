package ports

import "context"

type UploadStateClient interface {
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
	UpdateVideoMetadata(ctx context.Context, videoID, format string, durationSec float64, status string) error
}

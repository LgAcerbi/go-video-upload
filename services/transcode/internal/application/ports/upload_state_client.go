package ports

import "context"

type UploadStateClient interface {
	UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error
}

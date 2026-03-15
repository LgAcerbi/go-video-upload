package ports

import "context"

type UploadStepRepository interface {
	CreateSteps(ctx context.Context, uploadID string, steps []string) error
	UpdateStepStatus(ctx context.Context, uploadID, step, status, errorMessage string) error
}

package ports

import "context"

type StepTransitionResult struct {
	Applied        bool
	FromStatus     string
	ToStatus       string
	FailureReason  string
}

type UploadStepRepository interface {
	CreateSteps(ctx context.Context, uploadID string, steps []string) error
	UpdateStepStatus(ctx context.Context, uploadID, step, status, errorMessage string) (StepTransitionResult, error)
}

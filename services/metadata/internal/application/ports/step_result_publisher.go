package ports

import "context"

type StepResultPublisher interface {
	PublishStepResult(ctx context.Context, uploadID, step, status, errorMessage string) error
}

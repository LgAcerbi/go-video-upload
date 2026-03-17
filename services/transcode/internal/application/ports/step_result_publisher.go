package ports

import "context"

type StepResultPublisher interface {
	PublishStepResult(ctx context.Context, uploadID, videoID, step, status, errorMessage, storagePath string) error
}

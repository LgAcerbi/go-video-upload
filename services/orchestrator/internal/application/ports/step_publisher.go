package ports

import "context"

type StepPublisher interface {
	PublishStep(ctx context.Context, step, uploadID string) error
}

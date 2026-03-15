package ports

import "context"

type StepPublisher interface {
	PublishStep(ctx context.Context, step, videoID, uploadID, storagePath string) error
}

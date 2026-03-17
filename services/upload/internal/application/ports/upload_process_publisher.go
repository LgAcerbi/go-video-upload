package ports

import "context"

type UploadProcessPublisher interface {
	PublishUploadProcess(ctx context.Context, uploadID string) error
}

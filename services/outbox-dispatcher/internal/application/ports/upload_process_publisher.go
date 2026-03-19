package ports

import "context"

type UploadProcessPublisher interface {
	PublishUploadProcess(ctx context.Context, payload []byte) error
}

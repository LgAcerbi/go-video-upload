package ports

import "context"

type UploadProcessPublisher interface {
	PublishUploadProcess(ctx context.Context, videoID, uploadID, storagePath string) error
}

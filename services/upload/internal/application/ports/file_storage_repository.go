package ports

import (
	"context"
	"io"
	"time"
)

type UploadInput struct {
	Bucket        string
	Key           string
	Body          io.Reader
	ContentType   string
	ContentLength int64
}

type FileStorageRepository interface {
	Upload(ctx context.Context, input *UploadInput) error
	PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
}

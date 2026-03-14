package ports

import (
	"context"
	"io"
)

type UploadInput struct {
	Bucket        string
	Key           string
	Body          io.Reader
	ContentType   string
	ContentLength int64
}

type FileStorage interface {
	Upload(ctx context.Context, input *UploadInput) error
}

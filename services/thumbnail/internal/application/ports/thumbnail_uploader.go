package ports

import (
	"context"
	"io"
)

type ThumbnailUploader interface {
	UploadThumbnail(ctx context.Context, bucket, key string, body io.Reader, contentLength int64) error
}


package ports

import (
	"context"
	"io"
)

type RenditionUploader interface {
	UploadRendition(ctx context.Context, bucket, key string, body io.Reader, contentLength int64) error
}

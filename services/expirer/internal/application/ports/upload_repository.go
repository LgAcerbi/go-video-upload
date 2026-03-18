package ports

import (
	"context"
)

type UploadRepository interface {
	ExpireStaleUploads(ctx context.Context, limit int) (ExpireResult, error)
}

type ExpireResult struct {
	Found   int
	Expired int
	Skipped int
}


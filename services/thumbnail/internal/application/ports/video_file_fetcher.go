package ports

import "context"

type VideoFileFetcher interface {
	FetchToTempFile(ctx context.Context, bucket, key string) (path string, cleanup func(), err error)
}


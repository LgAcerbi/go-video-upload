package ports

import "context"

type ThumbnailGenerator interface {
	Generate(ctx context.Context, inputPath string) (outputPath string, cleanup func(), err error)
}


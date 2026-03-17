package ports

import "context"

type Transcoder interface {
	Transcode(ctx context.Context, inputPath string, height int) (outputPath string, cleanup func(), err error)
}

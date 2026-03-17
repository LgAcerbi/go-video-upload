package ports

import "context"

type DimensionsProber interface {
	Probe(ctx context.Context, filePath string) (width, height int, err error)
}

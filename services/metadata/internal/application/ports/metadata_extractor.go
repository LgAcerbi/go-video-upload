package ports

import "context"

type MetadataExtractor interface {
	Extract(ctx context.Context, filePath string) (format string, durationSec float64, width int32, height int32, err error)
}

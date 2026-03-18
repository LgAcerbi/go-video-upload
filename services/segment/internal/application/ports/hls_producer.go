package ports

import "context"

// HlsProducer turns an MP4 file (at inputPath) into HLS variant (playlist + segments) and uploads to object storage at the given prefix.
// outputPrefix is e.g. "videos/{videoID}/hls/720p" (no trailing slash).
type HlsProducer interface {
	ProduceAndUpload(ctx context.Context, bucket, outputPrefix, localMp4Path string) error
}

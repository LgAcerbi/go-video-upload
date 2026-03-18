package ports

import "context"

type MasterPlaylistUploader interface {
	UploadMasterPlaylist(ctx context.Context, bucket, key string, content []byte) error
}

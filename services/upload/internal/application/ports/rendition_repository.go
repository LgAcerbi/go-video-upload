package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type RenditionRepository interface {
	CreateBatch(ctx context.Context, videoID, originalStoragePath string, originalWidth, originalHeight int, targetHeights []int) error
	ListPendingByVideoID(ctx context.Context, videoID string) ([]*entities.Rendition, error)
	ListReadyByVideoID(ctx context.Context, videoID string) ([]*entities.Rendition, error)
	Update(ctx context.Context, r *entities.Rendition) error
}

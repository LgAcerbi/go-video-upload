package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type VideoRepository interface {
	Create(ctx context.Context, v *entities.Video) error
	GetByID(ctx context.Context, id string) (*entities.Video, error)
	Update(ctx context.Context, v *entities.Video) error
}

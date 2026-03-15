package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
)

type VideoRepository interface {
	Create(ctx context.Context, v *domain.Video) error
	GetByID(ctx context.Context, id string) (*domain.Video, error)
	Update(ctx context.Context, v *domain.Video) error
}

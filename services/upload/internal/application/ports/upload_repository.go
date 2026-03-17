package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type UploadRepository interface {
	Create(ctx context.Context, u *entities.Upload) error
	GetByVideoID(ctx context.Context, videoID string) (*entities.Upload, error)
	GetByID(ctx context.Context, uploadID string) (*entities.Upload, error)
	Update(ctx context.Context, u *entities.Upload) error
	UpdateStatus(ctx context.Context, uploadID, status string) error
	ListAll(ctx context.Context, limit int) ([]*entities.Upload, error)
}

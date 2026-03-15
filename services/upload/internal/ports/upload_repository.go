package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
)

type UploadRepository interface {
	Create(ctx context.Context, u *domain.Upload) error
	GetByVideoID(ctx context.Context, videoID string) (*domain.Upload, error)
	GetByID(ctx context.Context, uploadID string) (*domain.Upload, error)
	Update(ctx context.Context, u *domain.Upload) error
	UpdateStatus(ctx context.Context, uploadID, status string) error
}

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
	FinalizeProcessingWithOutbox(ctx context.Context, u *entities.Upload, eventType, idempotencyKey string, payload []byte) error
	UpdateStatus(ctx context.Context, uploadID, status string) error
	ListAll(ctx context.Context, limit int) ([]*entities.Upload, error)
	ListExpiredPending(ctx context.Context, limit int) ([]*entities.Upload, error)
	ExpireUploadAndSoftDeleteVideo(ctx context.Context, uploadID, videoID string) (bool, error)
}

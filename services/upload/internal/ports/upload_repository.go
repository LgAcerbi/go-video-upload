package ports

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
)

type UploadRepository interface {
	Create(ctx context.Context, u *domain.Upload) error
}

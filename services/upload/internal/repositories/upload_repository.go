package repository

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UploadRepository struct {
	pool *pgxpool.Pool
}

func NewUploadRepository(pool *pgxpool.Pool) ports.UploadRepository {
	return &UploadRepository{pool: pool}
}

func (r *UploadRepository) Create(ctx context.Context, u *domain.Upload) error {
	query := `
		INSERT INTO uploads (id, video_id, storage_path, status, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.VideoID, nullIfEmpty(u.StoragePath), u.Status, u.CreatedAt, u.UpdatedAt, u.ExpiresAt)
	return err
}

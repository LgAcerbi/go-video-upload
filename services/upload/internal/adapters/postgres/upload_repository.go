package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type UploadRepository struct {
	pool *pgxpool.Pool
}

func NewUploadRepository(pool *pgxpool.Pool) ports.UploadRepository {
	return &UploadRepository{pool: pool}
}

func (r *UploadRepository) Create(ctx context.Context, u *entities.Upload) error {
	query := `
		INSERT INTO uploads (id, video_id, storage_path, status, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query,
		u.ID, u.VideoID, nullIfEmpty(u.StoragePath), u.Status, u.CreatedAt, u.UpdatedAt, u.ExpiresAt)
	return err
}

func (r *UploadRepository) GetByVideoID(ctx context.Context, videoID string) (*entities.Upload, error) {
	query := `
		SELECT id, video_id, COALESCE(storage_path, ''), status, created_at, updated_at, deleted_at, expires_at
		FROM uploads WHERE video_id = $1 AND deleted_at IS NULL`
	var u entities.Upload
	err := r.pool.QueryRow(ctx, query, videoID).Scan(
		&u.ID, &u.VideoID, &u.StoragePath, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UploadRepository) GetByID(ctx context.Context, uploadID string) (*entities.Upload, error) {
	query := `
		SELECT id, video_id, COALESCE(storage_path, ''), status, created_at, updated_at, deleted_at, expires_at
		FROM uploads WHERE id = $1 AND deleted_at IS NULL`
	var u entities.Upload
	err := r.pool.QueryRow(ctx, query, uploadID).Scan(
		&u.ID, &u.VideoID, &u.StoragePath, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UploadRepository) Update(ctx context.Context, u *entities.Upload) error {
	query := `
		UPDATE uploads SET storage_path = $2, status = $3, updated_at = $4
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, u.ID, nullIfEmpty(u.StoragePath), u.Status, u.UpdatedAt)
	return err
}

func (r *UploadRepository) UpdateStatus(ctx context.Context, uploadID, status string) error {
	query := `UPDATE uploads SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, uploadID, status)
	return err
}

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/pkg/util"
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
		u.ID, u.VideoID, util.NullIfEmpty(u.StoragePath), u.Status, u.CreatedAt, u.UpdatedAt, u.ExpiresAt)
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
	_, err := r.pool.Exec(ctx, query, u.ID, util.NullIfEmpty(u.StoragePath), u.Status, u.UpdatedAt)
	return err
}

func (r *UploadRepository) UpdateStatus(ctx context.Context, uploadID, status string) error {
	query := `UPDATE uploads SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, uploadID, status)
	return err
}

const maxListLimit = 500

func (r *UploadRepository) ListAll(ctx context.Context, limit int) ([]*entities.Upload, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	query := `
		SELECT id, video_id, COALESCE(storage_path, ''), status, created_at, updated_at, deleted_at, expires_at
		FROM uploads WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1::integer`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Upload
	for rows.Next() {
		var u entities.Upload
		if err := rows.Scan(&u.ID, &u.VideoID, &u.StoragePath, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *UploadRepository) ListExpiredPending(ctx context.Context, limit int) ([]*entities.Upload, error) {
	if limit <= 0 {
		limit = 200
	}
	query := `
		SELECT id, video_id, COALESCE(storage_path, ''), status, created_at, updated_at, deleted_at, expires_at
		FROM uploads
		WHERE deleted_at IS NULL
		  AND status = 'pending'
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()
		ORDER BY expires_at ASC
		LIMIT $1::integer`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Upload
	for rows.Next() {
		var u entities.Upload
		if err := rows.Scan(&u.ID, &u.VideoID, &u.StoragePath, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, &u)
	}
	return out, rows.Err()
}

func (r *UploadRepository) ExpireUploadAndSoftDeleteVideo(ctx context.Context, uploadID, videoID string) (bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const expireUpload = `
		UPDATE uploads
		SET status = 'expired',
		    deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND video_id = $2
		  AND deleted_at IS NULL
		  AND status = 'pending'
		  AND expires_at IS NOT NULL
		  AND expires_at <= NOW()`
	tag, err := tx.Exec(ctx, expireUpload, uploadID, videoID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}

	const softDeleteVideo = `
		UPDATE videos
		SET status = 'failed',
		    deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, softDeleteVideo, videoID); err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

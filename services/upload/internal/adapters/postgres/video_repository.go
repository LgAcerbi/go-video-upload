package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/pkg/util"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type VideoRepository struct {
	pool *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) ports.VideoRepository {
	return &VideoRepository{pool: pool}
}

func (r *VideoRepository) Create(ctx context.Context, v *entities.Video) error {
	query := `
		INSERT INTO videos (id, user_id, title, format, status, duration_sec, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		v.ID, v.UserID, v.Title, util.NullIfEmpty(v.Format), v.Status, v.DurationSec, v.CreatedAt, v.UpdatedAt)
	return err
}

func (r *VideoRepository) GetByID(ctx context.Context, id string) (*entities.Video, error) {
	query := `SELECT id, user_id, title, COALESCE(format, ''), status, duration_sec, created_at, updated_at
		FROM videos WHERE id = $1 AND deleted_at IS NULL`
	var v entities.Video
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.UserID, &v.Title, &v.Format, &v.Status, &v.DurationSec, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) Update(ctx context.Context, v *entities.Video) error {
	query := `UPDATE videos SET format = $2, status = $3, duration_sec = $4, updated_at = $5 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, v.ID, util.NullIfEmpty(v.Format), v.Status, v.DurationSec, v.UpdatedAt)
	return err
}

func (r *VideoRepository) ListAll(ctx context.Context, limit int) ([]*entities.Video, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, user_id, title, COALESCE(format, ''), status, duration_sec, created_at, updated_at
		FROM videos WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1::integer`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Video
	for rows.Next() {
		var v entities.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Format, &v.Status, &v.DurationSec, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

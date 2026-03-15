package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
)

type UploadStepRepository struct {
	pool *pgxpool.Pool
}

func NewUploadStepRepository(pool *pgxpool.Pool) ports.UploadStepRepository {
	return &UploadStepRepository{pool: pool}
}

func (r *UploadStepRepository) CreateSteps(ctx context.Context, uploadID string, steps []string) error {
	for _, step := range steps {
		query := `
			INSERT INTO upload_steps (upload_id, step, status)
			VALUES ($1, $2, 'pending')
			ON CONFLICT (upload_id, step) DO NOTHING`
		_, err := r.pool.Exec(ctx, query, uploadID, step)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *UploadStepRepository) UpdateStepStatus(ctx context.Context, uploadID, step, status, errorMessage string) error {
	query := `
		UPDATE upload_steps SET status = $3, error_message = NULLIF($4, ''), updated_at = NOW()
		WHERE upload_id = $1 AND step = $2 AND status IN ('pending', 'processing')`
	_, err := r.pool.Exec(ctx, query, uploadID, step, status, errorMessage)
	return err
}

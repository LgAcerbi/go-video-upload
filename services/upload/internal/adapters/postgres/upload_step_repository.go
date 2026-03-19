package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
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

func (r *UploadStepRepository) UpdateStepStatus(ctx context.Context, uploadID, step, status, errorMessage string) (ports.StepTransitionResult, error) {
	result := ports.StepTransitionResult{
		Applied:    false,
		ToStatus:   status,
	}

	var query string
	switch status {
	case "processing":
		query = `
			UPDATE upload_steps
			SET status = $3, error_message = NULLIF($4, ''), updated_at = NOW()
			WHERE upload_id = $1 AND step = $2 AND status = 'pending'`
	case "done", "failed":
		query = `
			UPDATE upload_steps
			SET status = $3, error_message = NULLIF($4, ''), updated_at = NOW()
			WHERE upload_id = $1 AND step = $2 AND status = 'processing'`
	case "canceled":
		query = `
			UPDATE upload_steps
			SET status = $3, error_message = NULLIF($4, ''), updated_at = NOW()
			WHERE upload_id = $1 AND step = $2 AND status IN ('pending', 'processing')`
	default:
		result.FailureReason = "invalid_target_status"
		return result, nil
	}

	ct, err := r.pool.Exec(ctx, query, uploadID, step, status, errorMessage)
	if err != nil {
		return ports.StepTransitionResult{}, err
	}
	if ct.RowsAffected() == 1 {
		result.Applied = true
		return result, nil
	}

	const readStatusQuery = `
		SELECT status
		FROM upload_steps
		WHERE upload_id = $1 AND step = $2`
	var currentStatus string
	if err := r.pool.QueryRow(ctx, readStatusQuery, uploadID, step).Scan(&currentStatus); err != nil {
		if err == pgx.ErrNoRows {
			result.FailureReason = "step_not_found"
			return result, nil
		}
		return ports.StepTransitionResult{}, err
	}
	result.FromStatus = currentStatus
	result.FailureReason = "invalid_transition"
	return result, nil
}

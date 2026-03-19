package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/application/ports"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) ports.OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) ClaimPendingBatch(ctx context.Context, limit int) ([]ports.OutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
		WITH picked AS (
			SELECT id
			FROM outbox_events
			WHERE status = 'pending'
			  AND next_attempt_at <= NOW()
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events o
		SET status = 'dispatching',
		    attempts = o.attempts + 1,
		    last_error = NULL
		FROM picked
		WHERE o.id = picked.id
		RETURNING o.id, o.event_type, o.idempotency_key, o.payload::text, o.attempts`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ports.OutboxEvent
	for rows.Next() {
		var (
			ev          ports.OutboxEvent
			payloadText string
		)
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.Idempotency, &payloadText, &ev.Attempts); err != nil {
			return nil, err
		}
		ev.Payload = []byte(payloadText)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (r *OutboxRepository) MarkSent(ctx context.Context, eventID string) error {
	const query = `
		UPDATE outbox_events
		SET status = 'sent',
		    sent_at = NOW(),
		    last_error = NULL
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}

func (r *OutboxRepository) MarkPendingWithError(ctx context.Context, eventID, errMsg string, retryAfterSeconds int) error {
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	const query = `
		UPDATE outbox_events
		SET status = 'pending',
		    last_error = $2,
		    next_attempt_at = NOW() + ($3 * INTERVAL '1 second')
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, eventID, errMsg, retryAfterSeconds)
	return err
}

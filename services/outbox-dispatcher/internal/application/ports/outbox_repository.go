package ports

import "context"

type OutboxEvent struct {
	ID          string
	EventType   string
	Idempotency string
	Payload     []byte
	Attempts    int
}

type OutboxRepository interface {
	ClaimPendingBatch(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkSent(ctx context.Context, eventID string) error
	MarkPendingWithError(ctx context.Context, eventID, errMsg string, retryAfterSeconds int) error
}

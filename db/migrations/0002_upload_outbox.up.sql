CREATE TABLE IF NOT EXISTS outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts        INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_event_type_idempotency_key
    ON outbox_events (event_type, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_outbox_pending_next_attempt
    ON outbox_events (next_attempt_at, created_at)
    WHERE status = 'pending';

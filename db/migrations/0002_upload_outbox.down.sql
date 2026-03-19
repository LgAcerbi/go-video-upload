DROP INDEX IF EXISTS idx_outbox_pending_next_attempt;
DROP INDEX IF EXISTS idx_outbox_event_type_idempotency_key;
DROP TABLE IF EXISTS outbox_events;

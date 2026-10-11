CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY,
    request_event_id UUID NOT NULL UNIQUE,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    reason TEXT,
    payload JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    output_topic TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    retry_limit INTEGER NOT NULL CHECK (retry_limit >= 0),
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    status TEXT NOT NULL DEFAULT 'PENDING',
    processing_started_at TIMESTAMPTZ,
    last_attempt_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS outbox_pending_idx ON outbox (status, created_at)
    WHERE published_at IS NULL;

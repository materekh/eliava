CREATE TABLE reports (
    id          BIGSERIAL PRIMARY KEY,
    reporter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('post', 'listing', 'user')),
    target_id   BIGINT NOT NULL,
    reason      TEXT NOT NULL CHECK (reason IN ('fake', 'scam', 'unsafe', 'spam', 'other')),
    details     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    UNIQUE (reporter_id, target_type, target_id)
);
CREATE INDEX reports_open_idx ON reports (created_at DESC) WHERE status = 'open';

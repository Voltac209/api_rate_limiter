CREATE TABLE IF NOT EXISTS rate_limit_configs (
    key TEXT PRIMARY KEY,
    limit_value INTEGER NOT NULL CHECK (limit_value>0),
    window_seconds INTEGER NOT NULL CHECK (window_seconds>0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


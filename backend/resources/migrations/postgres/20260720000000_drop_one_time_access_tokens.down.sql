-- Recreate the one_time_access_tokens table with the schema it had before it was dropped
CREATE TABLE one_time_access_tokens
(
    id           UUID NOT NULL PRIMARY KEY,
    created_at   TIMESTAMPTZ,
    token        VARCHAR(255) NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    user_id      UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    device_token VARCHAR(16)
);
CREATE INDEX IF NOT EXISTS idx_one_time_access_tokens_expires_at ON one_time_access_tokens (expires_at);

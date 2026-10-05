CREATE TABLE known_browsers (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at BIGINT NOT NULL,
    PRIMARY KEY (user_id, token_hash)
);

CREATE INDEX idx_known_browsers_expires_at ON known_browsers (expires_at);

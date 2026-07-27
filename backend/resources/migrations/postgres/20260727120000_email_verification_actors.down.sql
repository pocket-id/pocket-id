CREATE TABLE email_verification_tokens
(
    id         UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    token      TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users ON DELETE CASCADE
);

CREATE INDEX idx_email_verification_tokens_expires_at ON email_verification_tokens (expires_at);

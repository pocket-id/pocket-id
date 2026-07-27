PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE email_verification_tokens
(
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id    TEXT     NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_email_verification_tokens_expires_at ON email_verification_tokens (expires_at);

COMMIT;
PRAGMA foreign_keys=ON;

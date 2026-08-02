PRAGMA foreign_keys = OFF;
BEGIN;

CREATE TABLE oauth2_sessions_old (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    request_id TEXT NOT NULL,
    access_token_signature TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    request_data BLOB NOT NULL,
    expires_at DATETIME
);

INSERT INTO oauth2_sessions_old (
    id,
    created_at,
    kind,
    key,
    request_id,
    access_token_signature,
    active,
    request_data,
    expires_at
)
SELECT
    id,
    created_at,
    kind,
    key,
    request_id,
    access_token_signature,
    active,
    request_data,
    expires_at
FROM oauth2_sessions;

DROP TABLE oauth2_sessions;
ALTER TABLE oauth2_sessions_old RENAME TO oauth2_sessions;

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);

COMMIT;
PRAGMA foreign_keys = ON;

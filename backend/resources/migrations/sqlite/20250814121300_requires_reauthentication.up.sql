PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN requires_reauthentication BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE reauthentication_tokens (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);

CREATE INDEX idx_reauthentication_tokens_token ON reauthentication_tokens(token);
COMMIT;
PRAGMA foreign_keys=ON;

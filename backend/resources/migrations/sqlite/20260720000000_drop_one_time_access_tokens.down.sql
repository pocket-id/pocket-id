PRAGMA foreign_keys=OFF;
BEGIN;

-- Recreate the one_time_access_tokens table with the schema it had before it was dropped
CREATE TABLE one_time_access_tokens
(
    id           TEXT PRIMARY KEY,
    created_at   DATETIME NOT NULL,
    token        TEXT NOT NULL UNIQUE,
    expires_at   DATETIME NOT NULL,
    user_id      TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    device_token TEXT
);
CREATE INDEX IF NOT EXISTS idx_one_time_access_tokens_expires_at ON one_time_access_tokens (expires_at);

COMMIT;
PRAGMA foreign_keys=ON;

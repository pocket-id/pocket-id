PRAGMA foreign_keys= OFF;
BEGIN;

CREATE TABLE recovery_codes
(
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    user_id    TEXT     NOT NULL,
    code_hash  TEXT     NOT NULL UNIQUE,
    used_at    DATETIME,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_recovery_codes_user_id ON recovery_codes (user_id);

COMMIT;
PRAGMA foreign_keys= ON;

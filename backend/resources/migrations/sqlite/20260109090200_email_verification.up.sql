PRAGMA foreign_keys= OFF;
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

ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET email_verified =EXISTS (SELECT 1
                            FROM app_config_variables
                            WHERE key = 'emailsVerified'
                              AND value = 'true');

COMMIT;
PRAGMA foreign_keys= ON;

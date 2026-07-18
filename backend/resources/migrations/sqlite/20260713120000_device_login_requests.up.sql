PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE device_login_requests
(
    id                TEXT     NOT NULL PRIMARY KEY,
    created_at        DATETIME NOT NULL,
    code              TEXT     NOT NULL UNIQUE,
    device_token_hash TEXT     NOT NULL UNIQUE,
    status            TEXT     NOT NULL CHECK (status IN ('pending', 'approved', 'denied')),
    expires_at        DATETIME NOT NULL,
    ip_address        TEXT     NOT NULL,
    user_agent        TEXT     NOT NULL,
    user_id           TEXT REFERENCES users ON DELETE CASCADE
);

CREATE INDEX idx_device_login_requests_expires_at ON device_login_requests (expires_at);

COMMIT;
PRAGMA foreign_keys=ON;

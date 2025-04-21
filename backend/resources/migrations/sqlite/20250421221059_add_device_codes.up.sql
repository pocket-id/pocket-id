CREATE TABLE oidc_device_codes
(
    id             TEXT     NOT NULL PRIMARY KEY,
    created_at     DATETIME,
    device_code    TEXT     NOT NULL UNIQUE,
    user_code      TEXT     NOT NULL UNIQUE,
    scope          TEXT     NOT NULL,
    expires_at     DATETIME NOT NULL,
    last_poll_time DATETIME,
    is_authorized  BOOLEAN  NOT NULL DEFAULT FALSE,
    user_id        TEXT REFERENCES users ON DELETE CASCADE,
    client_id      TEXT     NOT NULL REFERENCES oidc_clients ON DELETE CASCADE
);
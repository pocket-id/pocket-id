CREATE TABLE oidc_device_codes
(
    id             UUID        NOT NULL PRIMARY KEY,
    created_at     TIMESTAMPTZ,
    device_code    VARCHAR(255) NOT NULL UNIQUE,
    user_code      VARCHAR(255) NOT NULL UNIQUE,
    scope          TEXT        NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    interval       INTEGER     NOT NULL,
    last_poll_time TIMESTAMPTZ,
    is_authorized  BOOLEAN     NOT NULL DEFAULT FALSE,
    user_id        UUID REFERENCES users ON DELETE CASCADE,
    client_id      UUID        NOT NULL REFERENCES oidc_clients ON DELETE CASCADE
);
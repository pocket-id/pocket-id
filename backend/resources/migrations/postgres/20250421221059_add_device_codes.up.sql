CREATE TABLE oidc_device_codes
(
    id             UUID        NOT NULL PRIMARY KEY,
    created_at     TIMESTAMPTZ,
    device_code    TEXT        NOT NULL UNIQUE,
    user_code      TEXT        NOT NULL UNIQUE,
    scope          TEXT        NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL,
    is_authorized  BOOLEAN     NOT NULL DEFAULT FALSE,
    user_id        UUID REFERENCES users ON DELETE CASCADE,
    client_id      UUID        NOT NULL REFERENCES oidc_clients ON DELETE CASCADE
);

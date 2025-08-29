PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE users
(
    id         TEXT                  NOT NULL PRIMARY KEY,
    created_at DATETIME,
    username   TEXT                  NOT NULL UNIQUE,
    email      TEXT                  NOT NULL UNIQUE,
    first_name TEXT,
    last_name  TEXT,
    is_admin   NUMERIC DEFAULT FALSE NOT NULL
);

CREATE TABLE oidc_authorization_codes
(
    id         TEXT     NOT NULL PRIMARY KEY,
    created_at DATETIME,
    code       TEXT     NOT NULL UNIQUE,
    scope      TEXT     NOT NULL,
    nonce      TEXT,
    expires_at DATETIME NOT NULL,
    user_id    TEXT     NOT NULL REFERENCES users,
    client_id  TEXT     NOT NULL
);

CREATE TABLE oidc_clients
(
    id            TEXT NOT NULL PRIMARY KEY,
    created_at    DATETIME,
    name          TEXT,
    secret        TEXT,
    callback_url  TEXT,
    image_type    TEXT,
    created_by_id TEXT REFERENCES users
);

CREATE TABLE one_time_access_tokens
(
    id         TEXT     NOT NULL PRIMARY KEY,
    created_at DATETIME,
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id    TEXT     NOT NULL REFERENCES users
);

CREATE TABLE user_authorized_oidc_clients
(
    scope     TEXT,
    user_id   TEXT,
    client_id TEXT REFERENCES oidc_clients,
    PRIMARY KEY (user_id, client_id)
);

CREATE TABLE webauthn_credentials
(
    id               TEXT NOT NULL PRIMARY KEY,
    created_at       DATETIME,
    name             TEXT NOT NULL,
    credential_id    TEXT NOT NULL UNIQUE,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    transport        BLOB NOT NULL,
    user_id          TEXT REFERENCES users
);

CREATE TABLE webauthn_sessions
(
    id                TEXT     NOT NULL PRIMARY KEY,
    created_at        DATETIME,
    challenge         TEXT     NOT NULL UNIQUE,
    expires_at           DATETIME NOT NULL,
    user_verification TEXT     NOT NULL
);

CREATE TABLE application_configuration_variables
(
    key         TEXT                  NOT NULL PRIMARY KEY,
    value       TEXT                  NOT NULL,
    type        TEXT                  NOT NULL,
    is_public   NUMERIC DEFAULT FALSE NOT NULL,
    is_internal NUMERIC DEFAULT FALSE NOT NULL
);
COMMIT;
PRAGMA foreign_keys=ON;

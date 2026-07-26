PRAGMA foreign_keys = OFF;
BEGIN;

-- Align JSON and timestamp column types with the export format used for PostgreSQL
CREATE TABLE reauthentication_tokens_new (
    id TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);

INSERT INTO reauthentication_tokens_new (
    id,
    created_at,
    token,
    expires_at,
    user_id
)
SELECT
    id,
    created_at,
    token,
    expires_at,
    user_id
FROM reauthentication_tokens;

DROP TABLE reauthentication_tokens;
ALTER TABLE reauthentication_tokens_new RENAME TO reauthentication_tokens;

CREATE INDEX idx_reauthentication_tokens_token ON reauthentication_tokens (token);
CREATE INDEX idx_reauthentication_tokens_expires_at ON reauthentication_tokens (expires_at);

CREATE TABLE oauth2_sessions_new (
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

INSERT INTO oauth2_sessions_new (
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
    CAST(request_data AS BLOB),
    expires_at
FROM oauth2_sessions;

DROP TABLE oauth2_sessions;
ALTER TABLE oauth2_sessions_new RENAME TO oauth2_sessions;

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);

CREATE TABLE oauth2_jtis_new (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    jti TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL
);

INSERT INTO oauth2_jtis_new (
    id,
    created_at,
    jti,
    expires_at
)
SELECT
    id,
    created_at,
    jti,
    expires_at
FROM oauth2_jtis;

DROP TABLE oauth2_jtis;
ALTER TABLE oauth2_jtis_new RENAME TO oauth2_jtis;

CREATE INDEX idx_oauth2_jtis_expires_at ON oauth2_jtis (expires_at);

CREATE TABLE interaction_sessions_new (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    consent_required BOOLEAN NOT NULL DEFAULT FALSE,
    reauthentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    authentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    account_selection_required BOOLEAN NOT NULL DEFAULT FALSE,
    scopes BLOB NOT NULL DEFAULT X'5B5D',
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    requested_at DATETIME NOT NULL,
    reauthenticated_at DATETIME,
    parameters BLOB NOT NULL DEFAULT X'7B7D'
);

INSERT INTO interaction_sessions_new (
    id,
    created_at,
    consent_required,
    reauthentication_required,
    authentication_required,
    account_selection_required,
    scopes,
    client_id,
    user_id,
    requested_at,
    reauthenticated_at,
    parameters
)
SELECT
    id,
    created_at,
    consent_required,
    reauthentication_required,
    authentication_required,
    account_selection_required,
    CAST(scopes AS BLOB),
    client_id,
    user_id,
    requested_at,
    reauthenticated_at,
    CAST(parameters AS BLOB)
FROM interaction_sessions;

DROP TABLE interaction_sessions;
ALTER TABLE interaction_sessions_new RENAME TO interaction_sessions;

CREATE INDEX idx_interaction_sessions_client_id ON interaction_sessions (client_id);
CREATE INDEX idx_interaction_sessions_user_id ON interaction_sessions (user_id);

COMMIT;
PRAGMA foreign_keys = ON;

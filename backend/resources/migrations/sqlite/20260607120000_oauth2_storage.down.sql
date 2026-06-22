CREATE TABLE oidc_authorization_codes
(
    id                           TEXT NOT NULL PRIMARY KEY,
    created_at                   INTEGER,
    code                         TEXT NOT NULL UNIQUE,
    scope                        TEXT NOT NULL,
    nonce                        TEXT,
    expires_at                   INTEGER NOT NULL,
    user_id                      TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id                    TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    code_challenge               TEXT,
    code_challenge_method_sha256 NUMERIC,
    authentication_method        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_oidc_authorization_codes_expires_at ON oidc_authorization_codes (expires_at);

CREATE TABLE oidc_refresh_tokens (
    id TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER,
    token TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    scope TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    authentication_method TEXT NOT NULL DEFAULT '',
    id_token_jti TEXT
);

CREATE INDEX idx_oidc_refresh_tokens_expires_at ON oidc_refresh_tokens (expires_at);
CREATE INDEX idx_oidc_refresh_tokens_id_token_jti
    ON oidc_refresh_tokens(user_id, client_id, id_token_jti);

CREATE TABLE oidc_device_codes
(
    id                    TEXT    NOT NULL PRIMARY KEY,
    created_at            INTEGER,
    device_code           TEXT    NOT NULL UNIQUE,
    user_code             TEXT    NOT NULL UNIQUE,
    scope                 TEXT    NOT NULL,
    expires_at            INTEGER NOT NULL,
    is_authorized         BOOLEAN NOT NULL DEFAULT FALSE,
    user_id               TEXT REFERENCES users ON DELETE CASCADE,
    client_id             TEXT    NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    authentication_method TEXT    NOT NULL DEFAULT '',
    nonce                 TEXT
);

CREATE TABLE oidc_pushed_authorization_requests (
    id TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    request_uri TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    parameters TEXT NOT NULL DEFAULT '{}',
    expires_at INTEGER NOT NULL
);

CREATE INDEX idx_oidc_par_expires_at ON oidc_pushed_authorization_requests (expires_at);

CREATE TABLE user_authorized_oidc_clients_new (
    scope TEXT,
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    last_used_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, client_id)
);

INSERT INTO user_authorized_oidc_clients_new (scope, user_id, client_id, last_used_at)
SELECT CASE
        WHEN scope IS NULL THEN NULL
        WHEN json_valid(CAST(scope AS TEXT)) AND json_type(CAST(scope AS TEXT)) = 'array' THEN (
            SELECT group_concat(value, ' ')
            FROM json_each(CAST(scope AS TEXT))
        )
        ELSE CAST(scope AS TEXT)
    END,
    user_id,
    client_id,
    last_used_at
FROM user_authorized_oidc_clients;

DROP TABLE user_authorized_oidc_clients;
ALTER TABLE user_authorized_oidc_clients_new RENAME TO user_authorized_oidc_clients;

DROP INDEX IF EXISTS idx_interaction_sessions_client_id;
DROP INDEX IF EXISTS idx_interaction_sessions_user_id;
DROP TABLE IF EXISTS interaction_sessions;

DROP INDEX IF EXISTS idx_oauth2_jtis_expires_at;
DROP TABLE IF EXISTS oauth2_jtis;

DROP INDEX IF EXISTS idx_oauth2_sessions_expires_at;
DROP INDEX IF EXISTS idx_oauth2_sessions_kind_request;
DROP INDEX IF EXISTS idx_oauth2_sessions_kind_key;
DROP TABLE IF EXISTS oauth2_sessions;

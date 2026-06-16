CREATE TABLE oidc_authorization_codes
(
    id                           UUID NOT NULL PRIMARY KEY,
    created_at                   TIMESTAMPTZ,
    code                         VARCHAR(255) NOT NULL UNIQUE,
    scope                        TEXT NOT NULL,
    nonce                        VARCHAR(255),
    expires_at                   TIMESTAMPTZ NOT NULL,
    user_id                      UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id                    TEXT NOT NULL REFERENCES oidc_clients (id) ON DELETE CASCADE,
    code_challenge               VARCHAR(255),
    code_challenge_method_sha256 BOOLEAN,
    authentication_method        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_oidc_authorization_codes_expires_at ON oidc_authorization_codes (expires_at);

CREATE TABLE oidc_refresh_tokens (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    scope TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    authentication_method TEXT NOT NULL DEFAULT '',
    id_token_jti UUID
);

CREATE INDEX idx_oidc_refresh_tokens_expires_at ON oidc_refresh_tokens (expires_at);
CREATE INDEX idx_oidc_refresh_tokens_id_token_jti
    ON oidc_refresh_tokens(user_id, client_id, id_token_jti);

CREATE TABLE oidc_device_codes
(
    id                    UUID        NOT NULL PRIMARY KEY,
    created_at            TIMESTAMPTZ,
    device_code           TEXT        NOT NULL UNIQUE,
    user_code             TEXT        NOT NULL UNIQUE,
    scope                 TEXT        NOT NULL,
    expires_at            TIMESTAMPTZ NOT NULL,
    is_authorized         BOOLEAN     NOT NULL DEFAULT FALSE,
    user_id               UUID REFERENCES users ON DELETE CASCADE,
    client_id             TEXT        NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    authentication_method TEXT        NOT NULL DEFAULT '',
    nonce                 VARCHAR(255)
);

CREATE TABLE oidc_pushed_authorization_requests (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    request_uri TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    parameters JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oidc_par_expires_at ON oidc_pushed_authorization_requests (expires_at);

ALTER TABLE user_authorized_oidc_clients ADD COLUMN scope_text TEXT;
UPDATE user_authorized_oidc_clients
SET scope_text = CASE
    WHEN scope IS NULL THEN NULL
    WHEN jsonb_typeof(scope) = 'array' THEN (
        SELECT string_agg(scope_value.value, ' ')
        FROM jsonb_array_elements_text(scope) AS scope_value(value)
    )
    ELSE scope #>> '{}'
END;
ALTER TABLE user_authorized_oidc_clients DROP COLUMN scope;
ALTER TABLE user_authorized_oidc_clients RENAME COLUMN scope_text TO scope;

DROP INDEX IF EXISTS idx_interaction_sessions_client_id;
DROP INDEX IF EXISTS idx_interaction_sessions_user_id;
DROP TABLE IF EXISTS interaction_sessions;

DROP INDEX IF EXISTS idx_oauth2_jtis_expires_at;
DROP TABLE IF EXISTS oauth2_jtis;

DROP INDEX IF EXISTS idx_oauth2_sessions_expires_at;
DROP INDEX IF EXISTS idx_oauth2_sessions_kind_request;
DROP INDEX IF EXISTS idx_oauth2_sessions_kind_key;
DROP TABLE IF EXISTS oauth2_sessions;

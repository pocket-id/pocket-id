CREATE TABLE oauth2_sessions (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    request_id TEXT NOT NULL,
    access_token_signature TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    request_data JSONB NOT NULL,
    expires_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);

CREATE TABLE oauth2_jtis (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    jti TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth2_jtis_expires_at ON oauth2_jtis (expires_at);

CREATE TABLE interaction_sessions
(
    id                         UUID        NOT NULL PRIMARY KEY,
    created_at                 TIMESTAMPTZ NOT NULL,

    consent_required           BOOLEAN     NOT NULL DEFAULT FALSE,
    reauthentication_required  BOOLEAN     NOT NULL DEFAULT FALSE,
    authentication_required    BOOLEAN     NOT NULL DEFAULT FALSE,
    account_selection_required BOOLEAN     NOT NULL DEFAULT FALSE,

    scopes                     JSONB       NOT NULL DEFAULT '[]',
    client_id                  TEXT        NOT NULL REFERENCES oidc_clients (id) ON DELETE CASCADE,
    user_id                    UUID        REFERENCES users (id) ON DELETE CASCADE,
    requested_at               TIMESTAMPTZ NOT NULL,
    reauthenticated_at         TIMESTAMPTZ,
    parameters                 JSONB       NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_interaction_sessions_client_id
    ON interaction_sessions (client_id);
CREATE INDEX idx_interaction_sessions_user_id
    ON interaction_sessions (user_id);

-- Convert scope from string to json
ALTER TABLE user_authorized_oidc_clients
    ALTER COLUMN scope TYPE jsonb USING (
        CASE
            WHEN scope IS NULL OR btrim(scope) = '' THEN '[]'::jsonb
            ELSE to_jsonb(array_remove(regexp_split_to_array(scope, '[[:space:]]+'), ''))
        END
    ),
    ALTER COLUMN scope SET DEFAULT '[]'::jsonb,
    ALTER COLUMN scope SET NOT NULL;

DROP TABLE IF EXISTS oidc_pushed_authorization_requests;
DROP TABLE IF EXISTS oidc_device_codes;
DROP TABLE IF EXISTS oidc_refresh_tokens;
DROP TABLE IF EXISTS oidc_authorization_codes;

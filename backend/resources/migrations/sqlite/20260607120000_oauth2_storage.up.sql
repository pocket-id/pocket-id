PRAGMA foreign_keys= OFF;
BEGIN;

CREATE TABLE oauth2_sessions (
    id TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    request_id TEXT NOT NULL,
    access_token_signature TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    request_data TEXT NOT NULL,
    expires_at INTEGER
);

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);

CREATE TABLE oauth2_jtis (
    id TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    jti TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL
);

CREATE INDEX idx_oauth2_jtis_expires_at ON oauth2_jtis (expires_at);

CREATE TABLE interaction_sessions (
    id TEXT NOT NULL PRIMARY KEY,
    created_at INTEGER NOT NULL,
    consent_required BOOLEAN NOT NULL DEFAULT FALSE,
    reauthentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    authentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    account_selection_required BOOLEAN NOT NULL DEFAULT FALSE,
    scopes TEXT NOT NULL DEFAULT '[]',
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    requested_at INTEGER NOT NULL,
    reauthenticated_at INTEGER,
    parameters TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_interaction_sessions_client_id ON interaction_sessions (client_id);
CREATE INDEX idx_interaction_sessions_user_id ON interaction_sessions (user_id);

CREATE TABLE user_authorized_oidc_clients_new (
    scope BLOB NOT NULL DEFAULT X'5B5D',
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    last_used_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, client_id)
);

-- Convert scope from string to json
INSERT INTO user_authorized_oidc_clients_new (scope, user_id, client_id, last_used_at)
SELECT CAST(CASE
        WHEN scope IS NULL OR trim(CAST(scope AS TEXT)) = '' THEN '[]'
        ELSE (
            WITH RECURSIVE split(value, rest) AS (
                SELECT '', trim(CAST(scope AS TEXT)) || ' '
                UNION ALL
                SELECT substr(rest, 0, instr(rest, ' ')), ltrim(substr(rest, instr(rest, ' ') + 1))
                FROM split
                WHERE rest <> ''
            )
            SELECT json_group_array(value)
            FROM split
            WHERE value <> ''
        )
    END AS BLOB),
    user_id,
    client_id,
    last_used_at
FROM user_authorized_oidc_clients;

DROP TABLE user_authorized_oidc_clients;
ALTER TABLE user_authorized_oidc_clients_new RENAME TO user_authorized_oidc_clients;

DROP TABLE IF EXISTS oidc_pushed_authorization_requests;
DROP TABLE IF EXISTS oidc_device_codes;
DROP TABLE IF EXISTS oidc_refresh_tokens;
DROP TABLE IF EXISTS oidc_authorization_codes;

COMMIT;
PRAGMA foreign_keys= ON;


PRAGMA foreign_keys = OFF;
BEGIN;

-- Align the webauthn_sessions.extensions (JSONB) and oauth2_sessions.rotated_at (TIMESTAMPTZ) column types with the export format used for PostgreSQL
CREATE TABLE webauthn_sessions_new
(
    id                TEXT              NOT NULL PRIMARY KEY,
    created_at        DATETIME,
    challenge         TEXT              NOT NULL UNIQUE,
    expires_at        DATETIME          NOT NULL,
    user_verification TEXT              NOT NULL,
    credential_params BLOB DEFAULT '[]' NOT NULL,
    extensions        BLOB NOT NULL DEFAULT '{}'
);

INSERT INTO webauthn_sessions_new (
    id,
    created_at,
    challenge,
    expires_at,
    user_verification,
    credential_params,
    extensions
)
SELECT
    id,
    created_at,
    challenge,
    expires_at,
    user_verification,
    credential_params,
    extensions
FROM webauthn_sessions;

DROP TABLE webauthn_sessions;
ALTER TABLE webauthn_sessions_new RENAME TO webauthn_sessions;

CREATE INDEX idx_webauthn_sessions_expires_at ON webauthn_sessions (expires_at);

CREATE TABLE oauth2_sessions_new (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    request_id TEXT NOT NULL,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    access_token_signature TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    request_data BLOB NOT NULL,
    expires_at DATETIME,
    rotated_at DATETIME,
    CONSTRAINT chk_oauth2_sessions_client_id
        CHECK (client_id = json_extract(CAST(request_data AS TEXT), '$.client_id'))
);

INSERT INTO oauth2_sessions_new (
    id,
    created_at,
    kind,
    key,
    request_id,
    client_id,
    access_token_signature,
    active,
    request_data,
    expires_at,
    rotated_at
)
SELECT
    id,
    created_at,
    kind,
    key,
    request_id,
    client_id,
    access_token_signature,
    active,
    request_data,
    expires_at,
    rotated_at
FROM oauth2_sessions;

DROP TABLE oauth2_sessions;
ALTER TABLE oauth2_sessions_new RENAME TO oauth2_sessions;

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);
CREATE INDEX idx_oauth2_sessions_client_subject
    ON oauth2_sessions (client_id, json_extract(CAST(request_data AS TEXT), '$.session.subject'), kind, active);

-- Declare the TEXT primary keys as NOT NULL, because SQLite (unlike PostgreSQL) otherwise accepts NULL in them
-- Rows with a NULL id cannot be referenced or addressed by the application, so they are dropped

CREATE TABLE api_keys_new
(
    id                    TEXT NOT NULL PRIMARY KEY,
    name                  TEXT NOT NULL,
    key                   TEXT NOT NULL UNIQUE,
    description           TEXT,
    expires_at            DATETIME NOT NULL,
    last_used_at          DATETIME,
    created_at            DATETIME,
    user_id               TEXT REFERENCES users(id) ON DELETE CASCADE,
    expiration_email_sent BOOLEAN NOT NULL DEFAULT 0
);

INSERT INTO api_keys_new (
    id,
    name,
    key,
    description,
    expires_at,
    last_used_at,
    created_at,
    user_id,
    expiration_email_sent
)
SELECT
    id,
    name,
    key,
    description,
    expires_at,
    last_used_at,
    created_at,
    user_id,
    expiration_email_sent
FROM api_keys
WHERE id IS NOT NULL;

DROP TABLE api_keys;
ALTER TABLE api_keys_new RENAME TO api_keys;

CREATE INDEX idx_api_keys_key ON api_keys(key);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

CREATE TABLE audit_logs_new
(
    id         TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    event      TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT NOT NULL,
    data       BLOB NOT NULL,
    user_id    TEXT REFERENCES users ON DELETE CASCADE,
    country    TEXT,
    city       TEXT
);

INSERT INTO audit_logs_new (
    id,
    created_at,
    event,
    ip_address,
    user_agent,
    data,
    user_id,
    country,
    city
)
SELECT
    id,
    created_at,
    event,
    ip_address,
    user_agent,
    data,
    user_id,
    country,
    city
FROM audit_logs
WHERE id IS NOT NULL;

DROP TABLE audit_logs;
ALTER TABLE audit_logs_new RENAME TO audit_logs;

CREATE INDEX idx_audit_logs_client_name ON audit_logs((json_extract(data, '$.clientName')));
CREATE INDEX idx_audit_logs_country ON audit_logs (country);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_logs_event ON audit_logs (event);
CREATE INDEX idx_audit_logs_user_agent ON audit_logs (user_agent);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);

CREATE TABLE oidc_clients_new
(
    id                                     TEXT NOT NULL PRIMARY KEY,
    created_at                             DATETIME NOT NULL,
    name                                   TEXT,
    callback_urls                          BLOB,
    image_type                             TEXT,
    created_by_id                          TEXT REFERENCES users ON DELETE SET NULL,
    is_public                              BOOLEAN DEFAULT FALSE,
    pkce_enabled                           BOOLEAN DEFAULT FALSE,
    logout_callback_urls                   BLOB,
    credentials                            BLOB,
    launch_url                             TEXT,
    requires_reauthentication              BOOLEAN NOT NULL DEFAULT FALSE,
    dark_image_type                        TEXT,
    is_group_restricted                    BOOLEAN NOT NULL DEFAULT 0,
    requires_pushed_authorization_requests BOOLEAN NOT NULL DEFAULT FALSE,
    skip_consent                           BOOLEAN NOT NULL DEFAULT FALSE,
    pkce_supported                         BOOLEAN NOT NULL DEFAULT 0,
    description                            TEXT NOT NULL DEFAULT '',
    client_type                            TEXT NOT NULL DEFAULT 'standard',
    metadata_expires_at                    DATETIME,
    metadata_grant_types                   BLOB,
    access_token_duration_minutes          INTEGER NOT NULL DEFAULT 60,
    refresh_token_duration_minutes         INTEGER NOT NULL DEFAULT 43200,
    backchannel_logout_url                 TEXT NOT NULL DEFAULT ''
);

INSERT INTO oidc_clients_new (
    id,
    created_at,
    name,
    callback_urls,
    image_type,
    created_by_id,
    is_public,
    pkce_enabled,
    logout_callback_urls,
    credentials,
    launch_url,
    requires_reauthentication,
    dark_image_type,
    is_group_restricted,
    requires_pushed_authorization_requests,
    skip_consent,
    pkce_supported,
    description,
    client_type,
    metadata_expires_at,
    metadata_grant_types,
    access_token_duration_minutes,
    refresh_token_duration_minutes,
    backchannel_logout_url
)
SELECT
    id,
    created_at,
    name,
    callback_urls,
    image_type,
    created_by_id,
    is_public,
    pkce_enabled,
    logout_callback_urls,
    credentials,
    launch_url,
    requires_reauthentication,
    dark_image_type,
    is_group_restricted,
    requires_pushed_authorization_requests,
    skip_consent,
    pkce_supported,
    description,
    client_type,
    metadata_expires_at,
    metadata_grant_types,
    access_token_duration_minutes,
    refresh_token_duration_minutes,
    backchannel_logout_url
FROM oidc_clients
WHERE id IS NOT NULL;

DROP TABLE oidc_clients;
ALTER TABLE oidc_clients_new RENAME TO oidc_clients;

CREATE TABLE reauthentication_tokens_new
(
    id         TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users ON DELETE CASCADE
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
FROM reauthentication_tokens
WHERE id IS NOT NULL;

DROP TABLE reauthentication_tokens;
ALTER TABLE reauthentication_tokens_new RENAME TO reauthentication_tokens;

CREATE INDEX idx_reauthentication_tokens_token ON reauthentication_tokens (token);
CREATE INDEX idx_reauthentication_tokens_expires_at ON reauthentication_tokens (expires_at);

CREATE TABLE scim_service_providers_new
(
    id             TEXT NOT NULL PRIMARY KEY,
    created_at     DATETIME NOT NULL,
    endpoint       TEXT NOT NULL,
    token          TEXT NOT NULL,
    last_synced_at DATETIME,
    oidc_client_id TEXT NOT NULL,
    FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE
);

INSERT INTO scim_service_providers_new (
    id,
    created_at,
    endpoint,
    token,
    last_synced_at,
    oidc_client_id
)
SELECT
    id,
    created_at,
    endpoint,
    token,
    last_synced_at,
    oidc_client_id
FROM scim_service_providers
WHERE id IS NOT NULL;

DROP TABLE scim_service_providers;
ALTER TABLE scim_service_providers_new RENAME TO scim_service_providers;

CREATE TABLE webauthn_credentials_new
(
    id               TEXT NOT NULL PRIMARY KEY,
    created_at       DATETIME NOT NULL,
    name             TEXT NOT NULL,
    credential_id    BLOB NOT NULL UNIQUE,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    transport        BLOB NOT NULL,
    user_id          TEXT REFERENCES users ON DELETE CASCADE,
    backup_eligible  BOOLEAN DEFAULT FALSE NOT NULL,
    backup_state     BOOLEAN DEFAULT FALSE NOT NULL,
    aaguid           TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'
);

INSERT INTO webauthn_credentials_new (
    id,
    created_at,
    name,
    credential_id,
    public_key,
    attestation_type,
    transport,
    user_id,
    backup_eligible,
    backup_state,
    aaguid
)
SELECT
    id,
    created_at,
    name,
    credential_id,
    public_key,
    attestation_type,
    transport,
    user_id,
    backup_eligible,
    backup_state,
    aaguid
FROM webauthn_credentials
WHERE id IS NOT NULL;

DROP TABLE webauthn_credentials;
ALTER TABLE webauthn_credentials_new RENAME TO webauthn_credentials;

COMMIT;
PRAGMA foreign_keys = ON;

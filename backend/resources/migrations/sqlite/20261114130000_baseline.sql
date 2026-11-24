-- +goose Up

CREATE TABLE user_groups
(
    id           TEXT NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    friendly_name TEXT NOT NULL,
    name        TEXT NOT NULL UNIQUE
, ldap_id TEXT, updated_at DATETIME, notes TEXT NOT NULL DEFAULT '');

CREATE TABLE user_groups_users
(
    user_id  TEXT NOT NULL,
    user_group_id TEXT NOT NULL,
    PRIMARY KEY (user_id, user_group_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);

CREATE TABLE custom_claims
(
    id            TEXT NOT NULL PRIMARY KEY,
    created_at    DATETIME,
    key           TEXT NOT NULL,
    value         TEXT NOT NULL,

    user_id       TEXT,
    user_group_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE,

    CONSTRAINT custom_claims_unique UNIQUE (key, user_id, user_group_id),
    CHECK (user_id IS NOT NULL OR user_group_id IS NOT NULL)
);

CREATE TABLE oidc_clients_allowed_user_groups
(
    user_group_id  TEXT NOT NULL,
    oidc_client_id TEXT NOT NULL,
    PRIMARY KEY (oidc_client_id, user_group_id),
    FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE,
    FOREIGN KEY (user_group_id) REFERENCES user_groups (id) ON DELETE CASCADE
);

CREATE TABLE kv
(
    "key"  TEXT NOT NULL PRIMARY KEY,
    "value" TEXT NOT NULL
);

CREATE TABLE storage
(
    path       TEXT NOT NULL PRIMARY KEY,
    data       BLOB NOT NULL,
    size       INTEGER NOT NULL,
    mod_time   DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE "users"
(
    id                          TEXT                  NOT NULL PRIMARY KEY,
    created_at                  DATETIME,
    updated_at                  DATETIME,
    username                    TEXT COLLATE NOCASE   NOT NULL UNIQUE,
    email                       TEXT COLLATE NOCASE UNIQUE,
    first_name                  TEXT                  NOT NULL,
    last_name                   TEXT                  NOT NULL,
    display_name                TEXT                  NOT NULL,
    is_admin                    BOOLEAN DEFAULT FALSE NOT NULL,
    ldap_id                     TEXT UNIQUE,
    locale                      TEXT,
    disabled                    BOOLEAN DEFAULT FALSE NOT NULL,
    email_verified              BOOLEAN DEFAULT FALSE NOT NULL,
    email_verification_deadline DATETIME
);

CREATE TABLE "user_authorized_oidc_clients" (
    scope BLOB NOT NULL DEFAULT X'5B5D',
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    last_used_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, client_id)
);

CREATE TABLE apis (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    name TEXT NOT NULL,
    audience TEXT NOT NULL UNIQUE
, allow_cimd_clients BOOLEAN NOT NULL DEFAULT FALSE);

CREATE TABLE api_permissions (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    api_id TEXT NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT, allowed_for_cimd_clients BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (api_id, key)
);

CREATE TABLE oidc_clients_allowed_api_permissions (
    oidc_client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    api_permission_id TEXT NOT NULL REFERENCES api_permissions(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL CHECK (subject_type IN ('user', 'client')),
    PRIMARY KEY (oidc_client_id, api_permission_id, subject_type)
);

CREATE TABLE "oauth2_jtis" (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    jti TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL
);

CREATE TABLE "interaction_sessions" (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    consent_required BOOLEAN NOT NULL DEFAULT FALSE,
    reauthentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    authentication_required BOOLEAN NOT NULL DEFAULT FALSE,
    account_selection_required BOOLEAN NOT NULL DEFAULT FALSE,
    scopes BLOB NOT NULL DEFAULT X'5B5D',
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    requested_at DATETIME NOT NULL,
    reauthenticated_at DATETIME,
    parameters BLOB NOT NULL DEFAULT X'7B7D',
    session_id TEXT
);

CREATE TABLE oidc_clients_allowed_apis (
    oidc_client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    api_id TEXT NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    subject_type TEXT NOT NULL CHECK (subject_type IN ('user', 'client')),
    PRIMARY KEY (oidc_client_id, api_id, subject_type)
);

CREATE TABLE "webauthn_sessions"
(
    id                TEXT              NOT NULL PRIMARY KEY,
    created_at        DATETIME,
    challenge         TEXT              NOT NULL UNIQUE,
    expires_at        DATETIME          NOT NULL,
    user_verification TEXT              NOT NULL,
    credential_params BLOB DEFAULT '[]' NOT NULL,
    extensions        BLOB NOT NULL DEFAULT '{}'
);

CREATE TABLE "oauth2_sessions" (
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

CREATE TABLE "api_keys"
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
, scopes BLOB NOT NULL DEFAULT X'5B5D', previous_key TEXT, previous_key_expires_at DATETIME);

CREATE TABLE "oidc_clients"
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
    notes                            TEXT NOT NULL DEFAULT '',
    client_type                            TEXT NOT NULL DEFAULT 'standard',
    metadata_expires_at                    DATETIME,
    metadata_grant_types                   BLOB,
    access_token_duration_minutes          INTEGER NOT NULL DEFAULT 60,
    refresh_token_duration_minutes         INTEGER NOT NULL DEFAULT 43200,
    backchannel_logout_url                 TEXT NOT NULL DEFAULT ''
, claim_mapping_policy_id TEXT REFERENCES oidc_claim_mapping_policies(id) ON DELETE SET NULL, disabled BOOLEAN NOT NULL DEFAULT FALSE);

CREATE TABLE "reauthentication_tokens"
(
    id         TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);

CREATE TABLE "scim_service_providers"
(
    id             TEXT NOT NULL PRIMARY KEY,
    created_at     DATETIME NOT NULL,
    endpoint       TEXT NOT NULL,
    token          TEXT NOT NULL,
    last_synced_at DATETIME,
    oidc_client_id TEXT NOT NULL,
    FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE
);

CREATE TABLE "webauthn_credentials"
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
, attestation_verified BOOLEAN NOT NULL DEFAULT FALSE);

CREATE TABLE oidc_claim_mapping_policies (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    claim_mappings BLOB NOT NULL
);

CREATE TABLE "audit_logs"
(
    id         TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    event      TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT NOT NULL,
    data       BLOB NOT NULL,
    user_id    TEXT REFERENCES users ON DELETE SET NULL,
    country    TEXT,
    city       TEXT,
    actor_type TEXT,
    actor_id   TEXT,
    actor_name TEXT
);

CREATE TABLE oidc_custom_scopes (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE UNIQUE INDEX user_groups_ldap_id ON user_groups (ldap_id);

CREATE INDEX idx_api_permissions_api_id ON api_permissions(api_id);

CREATE INDEX idx_oauth2_jtis_expires_at ON oauth2_jtis (expires_at);

CREATE INDEX idx_interaction_sessions_client_id ON interaction_sessions (client_id);

CREATE INDEX idx_oidc_clients_allowed_apis_api_id ON oidc_clients_allowed_apis(api_id);

CREATE INDEX idx_webauthn_sessions_expires_at ON webauthn_sessions (expires_at);

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);

CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id, active, rotated_at);

CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);

CREATE INDEX idx_oauth2_sessions_client_subject
    ON oauth2_sessions (client_id, json_extract(CAST(request_data AS TEXT), '$.session.subject'), kind, active);

CREATE INDEX idx_api_keys_key ON api_keys(key);

CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

CREATE INDEX idx_reauthentication_tokens_token ON reauthentication_tokens (token);

CREATE INDEX idx_reauthentication_tokens_expires_at ON reauthentication_tokens (expires_at);

CREATE UNIQUE INDEX idx_oidc_claim_mapping_policies_default ON oidc_claim_mapping_policies (is_default) WHERE is_default;

CREATE UNIQUE INDEX idx_api_keys_previous_key ON api_keys(previous_key);

CREATE INDEX idx_audit_logs_client_name ON audit_logs((json_extract(data, '$.clientName')));

CREATE INDEX idx_audit_logs_country ON audit_logs (country);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

CREATE INDEX idx_audit_logs_event ON audit_logs (event);

CREATE INDEX idx_audit_logs_user_agent ON audit_logs (user_agent);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);

CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id);

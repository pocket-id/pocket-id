PRAGMA foreign_keys=OFF;
---------------------------
-- Delete all orphaned rows
---------------------------
UPDATE oidc_clients
SET created_by_id = NULL
WHERE created_by_id IS NOT NULL
  AND created_by_id NOT IN (SELECT id FROM users);

DELETE FROM oidc_authorization_codes WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM one_time_access_tokens WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM webauthn_credentials WHERE user_id NOT IN (SELECT id FROM users);
DELETE FROM audit_logs WHERE user_id IS NOT NULL AND user_id NOT IN (SELECT id FROM users);
DELETE FROM api_keys WHERE user_id IS NOT NULL AND user_id NOT IN (SELECT id FROM users);

DELETE FROM oidc_refresh_tokens WHERE user_id NOT IN (SELECT id FROM users) OR client_id NOT IN (SELECT id FROM oidc_clients);
DELETE FROM oidc_device_codes WHERE (user_id IS NOT NULL AND user_id NOT IN (SELECT id FROM users)) OR client_id NOT IN (SELECT id FROM oidc_clients);
DELETE FROM user_authorized_oidc_clients WHERE user_id NOT IN (SELECT id FROM users) OR client_id NOT IN (SELECT id FROM oidc_clients);

DELETE FROM user_groups_users WHERE user_id NOT IN (SELECT id FROM users) OR user_group_id NOT IN (SELECT id FROM user_groups);

DELETE FROM custom_claims WHERE (user_id IS NOT NULL AND user_id NOT IN (SELECT id FROM users)) OR (user_group_id IS NOT NULL AND user_group_id NOT IN (SELECT id FROM user_groups));

DELETE FROM oidc_clients_allowed_user_groups WHERE oidc_client_id NOT IN (SELECT id FROM oidc_clients) OR user_group_id NOT IN (SELECT id FROM user_groups);

DELETE FROM reauthentication_tokens WHERE user_id NOT IN (SELECT id FROM users);

---------------------------
-- Add missing foreign keys and edit cascade behavior where necessary
---------------------------

-- reauthentication_tokens: add missing FK user_id → users
CREATE TABLE reauthentication_tokens_new
(
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);
INSERT INTO reauthentication_tokens_new (id, created_at, token, expires_at, user_id)
SELECT id, created_at, token, expires_at, user_id
FROM reauthentication_tokens;
DROP TABLE reauthentication_tokens;
ALTER TABLE reauthentication_tokens_new RENAME TO reauthentication_tokens;
CREATE INDEX idx_reauthentication_tokens_token
    ON reauthentication_tokens (token);

-- oidc_authorization_codes: add FK client_id, user_id → CASCADE
CREATE TABLE oidc_authorization_codes_new
(
    id                           TEXT PRIMARY KEY,
    created_at                   DATETIME NOT NULL,
    code                         TEXT NOT NULL UNIQUE,
    scope                        TEXT NOT NULL,
    nonce                        TEXT,
    expires_at                   DATETIME NOT NULL,
    user_id                      TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id                    TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    code_challenge               TEXT,
    code_challenge_method_sha256 NUMERIC
);
INSERT INTO oidc_authorization_codes_new
    (id, created_at, code, scope, nonce, expires_at, user_id, client_id, code_challenge, code_challenge_method_sha256)
SELECT id, created_at, code, scope, nonce, expires_at, user_id, client_id, code_challenge, code_challenge_method_sha256
FROM oidc_authorization_codes;
DROP TABLE oidc_authorization_codes;
ALTER TABLE oidc_authorization_codes_new RENAME TO oidc_authorization_codes;

-- user_authorized_oidc_clients: add FK user_id, cascade client_id
CREATE TABLE user_authorized_oidc_clients_new
(
    scope        TEXT,
    user_id      TEXT NOT NULL REFERENCES users ON DELETE CASCADE,
    client_id    TEXT NOT NULL REFERENCES oidc_clients ON DELETE CASCADE,
    last_used_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, client_id)
);
INSERT INTO user_authorized_oidc_clients_new (scope, user_id, client_id, last_used_at)
SELECT scope, user_id, client_id, last_used_at
FROM user_authorized_oidc_clients;
DROP TABLE user_authorized_oidc_clients;
ALTER TABLE user_authorized_oidc_clients_new RENAME TO user_authorized_oidc_clients;

-- audit_logs: user_id → CASCADE
CREATE TABLE audit_logs_new
(
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    event      TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT NOT NULL,
    data       BLOB NOT NULL,
    user_id    TEXT REFERENCES users ON DELETE CASCADE,
    country    TEXT,
    city       TEXT
);
INSERT INTO audit_logs_new
    (id, created_at, event, ip_address, user_agent, data, user_id, country, city)
SELECT id, created_at, event, ip_address, user_agent, data, user_id, country, city
FROM audit_logs;
DROP TABLE audit_logs;
ALTER TABLE audit_logs_new RENAME TO audit_logs;
CREATE INDEX idx_audit_logs_client_name ON audit_logs((json_extract(data, '$.clientName')));
CREATE INDEX idx_audit_logs_country ON audit_logs (country);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_logs_event ON audit_logs (event);
CREATE INDEX idx_audit_logs_user_agent ON audit_logs (user_agent);
CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);

-- oidc_clients: created_by_id → SET NULL
CREATE TABLE oidc_clients_new
(
    id                        TEXT PRIMARY KEY,
    created_at                DATETIME NOT NULL,
    name                      TEXT,
    secret                    TEXT,
    callback_urls             BLOB,
    image_type                TEXT,
    created_by_id             TEXT REFERENCES users ON DELETE SET NULL,
    is_public                 BOOLEAN DEFAULT FALSE,
    pkce_enabled              BOOLEAN DEFAULT FALSE,
    logout_callback_urls      BLOB,
    credentials               TEXT,
    launch_url                TEXT,
    requires_reauthentication BOOLEAN DEFAULT FALSE NOT NULL
);
INSERT INTO oidc_clients_new
    (id, created_at, name, secret, callback_urls, image_type, created_by_id,
     is_public, pkce_enabled, logout_callback_urls, credentials, launch_url, requires_reauthentication)
SELECT id, created_at, name, secret, callback_urls, image_type, created_by_id,
       is_public, pkce_enabled, logout_callback_urls, credentials, launch_url, requires_reauthentication
FROM oidc_clients;
DROP TABLE oidc_clients;
ALTER TABLE oidc_clients_new RENAME TO oidc_clients;

-- one_time_access_tokens: user_id → CASCADE
CREATE TABLE one_time_access_tokens_new
(
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token      TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);
INSERT INTO one_time_access_tokens_new
    (id, created_at, token, expires_at, user_id)
SELECT id, created_at, token, expires_at, user_id
FROM one_time_access_tokens;
DROP TABLE one_time_access_tokens;
ALTER TABLE one_time_access_tokens_new RENAME TO one_time_access_tokens;

-- webauthn_credentials: user_id → CASCADE
CREATE TABLE webauthn_credentials_new
(
    id               TEXT PRIMARY KEY,
    created_at       DATETIME NOT NULL,
    name             TEXT NOT NULL,
    credential_id    TEXT NOT NULL UNIQUE,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL,
    transport        BLOB NOT NULL,
    user_id          TEXT REFERENCES users ON DELETE CASCADE,
    backup_eligible  BOOLEAN DEFAULT FALSE NOT NULL,
    backup_state     BOOLEAN DEFAULT FALSE NOT NULL
);
INSERT INTO webauthn_credentials_new
    (id, created_at, name, credential_id, public_key, attestation_type,
     transport, user_id, backup_eligible, backup_state)
SELECT id, created_at, name, credential_id, public_key, attestation_type,
       transport, user_id, backup_eligible, backup_state
FROM webauthn_credentials;
DROP TABLE webauthn_credentials;
ALTER TABLE webauthn_credentials_new RENAME TO webauthn_credentials;

PRAGMA foreign_keys=ON;
PRAGMA foreign_key_check;
PRAGMA foreign_keys=OFF;
BEGIN;
-- Convert the DATETIME fields to Unix timestamps (in seconds)
UPDATE user_groups
SET created_at = strftime('%s', created_at);

UPDATE users
SET created_at = strftime('%s', created_at);

UPDATE audit_logs
SET created_at = strftime('%s', created_at);

UPDATE oidc_authorization_codes
SET created_at = strftime('%s', created_at),
    expires_at = strftime('%s', expires_at);

UPDATE oidc_clients
SET created_at = strftime('%s', created_at);

UPDATE one_time_access_tokens
SET created_at = strftime('%s', created_at),
    expires_at = strftime('%s', expires_at);

UPDATE webauthn_credentials
SET created_at = strftime('%s', created_at);

UPDATE webauthn_sessions
SET created_at = strftime('%s', created_at),
    expires_at = strftime('%s', expires_at);
COMMIT;
PRAGMA foreign_keys=ON;

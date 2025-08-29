PRAGMA foreign_keys=OFF;
BEGIN;
-- Convert the Unix timestamps back to DATETIME format

UPDATE user_groups
SET created_at = datetime(created_at, 'unixepoch');

UPDATE users
SET created_at = datetime(created_at, 'unixepoch');

UPDATE audit_logs
SET created_at = datetime(created_at, 'unixepoch');

UPDATE oidc_authorization_codes
SET created_at = datetime(created_at, 'unixepoch'),
    expires_at = datetime(expires_at, 'unixepoch');

UPDATE oidc_clients
SET created_at = datetime(created_at, 'unixepoch');

UPDATE one_time_access_tokens
SET created_at = datetime(created_at, 'unixepoch'),
    expires_at = datetime(expires_at, 'unixepoch');

UPDATE webauthn_credentials
SET created_at = datetime(created_at, 'unixepoch');

UPDATE webauthn_sessions
SET created_at = datetime(created_at, 'unixepoch'),
    expires_at = datetime(expires_at, 'unixepoch');
COMMIT;
PRAGMA foreign_keys=ON;

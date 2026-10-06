-- Declare the columns that are NOT NULL on SQLite as NOT NULL too, so an export from either database can be imported into the other
-- The application never writes NULL into these columns, but backfill any NULL left over from older versions first
-- Empty strings are what the application writes for these values and what readers of kv treat as missing

UPDATE audit_logs SET created_at = now() WHERE created_at IS NULL;
ALTER TABLE audit_logs ALTER COLUMN created_at SET NOT NULL;

UPDATE audit_logs SET user_agent = '' WHERE user_agent IS NULL;
ALTER TABLE audit_logs ALTER COLUMN user_agent SET NOT NULL;

UPDATE kv SET value = '' WHERE value IS NULL;
ALTER TABLE kv ALTER COLUMN value SET NOT NULL;

UPDATE oidc_clients SET created_at = now() WHERE created_at IS NULL;
ALTER TABLE oidc_clients ALTER COLUMN created_at SET NOT NULL;

UPDATE users SET first_name = '' WHERE first_name IS NULL;
ALTER TABLE users ALTER COLUMN first_name SET NOT NULL;

UPDATE users SET last_name = '' WHERE last_name IS NULL;
ALTER TABLE users ALTER COLUMN last_name SET NOT NULL;

UPDATE webauthn_credentials SET created_at = now() WHERE created_at IS NULL;
ALTER TABLE webauthn_credentials ALTER COLUMN created_at SET NOT NULL;

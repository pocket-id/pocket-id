ALTER TABLE oidc_clients DROP COLUMN requires_reauthentication;
DROP INDEX IF EXISTS idx_reauthentication_tokens_token;
DROP TABLE IF EXISTS reauthentication_tokens;
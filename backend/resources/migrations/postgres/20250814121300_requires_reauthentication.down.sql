ALTER TABLE oidc_clients DROP COLUMN requires_reauthentication;
DROP TABLE IF EXISTS reauthentication_tokens;

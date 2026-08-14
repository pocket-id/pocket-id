PRAGMA foreign_keys = OFF;
BEGIN;

DROP TRIGGER IF EXISTS prevent_runtime_credential_on_passkey_path;
DROP TRIGGER IF EXISTS prevent_passkey_on_runtime_path;
DROP TRIGGER IF EXISTS prevent_agent_identifier_change_with_credentials;
DROP TABLE IF EXISTS runtime_credential_challenges;
DROP TABLE IF EXISTS runtime_credentials;
DROP INDEX IF EXISTS idx_users_agent_identifier;
ALTER TABLE users DROP COLUMN agent_identifier;

COMMIT;
PRAGMA foreign_keys = ON;

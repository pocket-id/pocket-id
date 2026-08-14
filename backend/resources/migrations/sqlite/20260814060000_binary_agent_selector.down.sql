PRAGMA foreign_keys = OFF;
BEGIN;

DROP TRIGGER IF EXISTS prevent_is_agent_change_with_credentials;
DROP TRIGGER IF EXISTS prevent_passkey_on_runtime_path;
DROP TRIGGER IF EXISTS prevent_runtime_credential_on_passkey_path;

ALTER TABLE users ADD COLUMN agent_identifier TEXT;
UPDATE users SET agent_identifier = id WHERE is_agent = 1;
ALTER TABLE users DROP COLUMN is_agent;

CREATE UNIQUE INDEX idx_users_agent_identifier ON users (agent_identifier);

CREATE TRIGGER prevent_agent_identifier_change_with_credentials
BEFORE UPDATE OF agent_identifier ON users
WHEN OLD.agent_identifier IS NOT NEW.agent_identifier
 AND (
    EXISTS (SELECT 1 FROM webauthn_credentials WHERE user_id = OLD.id)
    OR EXISTS (SELECT 1 FROM runtime_credentials WHERE user_id = OLD.id AND revoked_at IS NULL)
 )
BEGIN
    SELECT RAISE(ABORT, 'authentication path cannot change while credentials exist');
END;

CREATE TRIGGER prevent_passkey_on_runtime_path
BEFORE INSERT ON webauthn_credentials
WHEN EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND agent_identifier IS NOT NULL)
BEGIN
    SELECT RAISE(ABORT, 'passkeys are not allowed on the runtime authentication path');
END;

CREATE TRIGGER prevent_runtime_credential_on_passkey_path
BEFORE INSERT ON runtime_credentials
WHEN EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND agent_identifier IS NULL)
BEGIN
    SELECT RAISE(ABORT, 'runtime credentials require the runtime authentication path');
END;

COMMIT;
PRAGMA foreign_keys = ON;

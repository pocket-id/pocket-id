PRAGMA foreign_keys = OFF;
BEGIN;

DROP TRIGGER IF EXISTS prevent_agent_identifier_change_with_credentials;
DROP TRIGGER IF EXISTS prevent_passkey_on_runtime_path;
DROP TRIGGER IF EXISTS prevent_runtime_credential_on_passkey_path;
DROP INDEX IF EXISTS idx_users_agent_identifier;

ALTER TABLE users ADD COLUMN is_agent BOOLEAN NOT NULL DEFAULT 0;
UPDATE users SET is_agent = CASE WHEN agent_identifier IS NULL THEN 0 ELSE 1 END;
ALTER TABLE users DROP COLUMN agent_identifier;

CREATE TRIGGER prevent_is_agent_change_with_credentials
BEFORE UPDATE OF is_agent ON users
WHEN OLD.is_agent IS NOT NEW.is_agent
 AND (
    EXISTS (SELECT 1 FROM webauthn_credentials WHERE user_id = OLD.id)
    OR EXISTS (SELECT 1 FROM runtime_credentials WHERE user_id = OLD.id AND revoked_at IS NULL)
 )
BEGIN
    SELECT RAISE(ABORT, 'authentication path cannot change while credentials exist');
END;

CREATE TRIGGER prevent_passkey_on_runtime_path
BEFORE INSERT ON webauthn_credentials
WHEN EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND is_agent = 1)
BEGIN
    SELECT RAISE(ABORT, 'passkeys are not allowed on the runtime authentication path');
END;

CREATE TRIGGER prevent_runtime_credential_on_passkey_path
BEFORE INSERT ON runtime_credentials
WHEN EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND is_agent = 0)
BEGIN
    SELECT RAISE(ABORT, 'runtime credentials require the runtime authentication path');
END;

COMMIT;
PRAGMA foreign_keys = ON;

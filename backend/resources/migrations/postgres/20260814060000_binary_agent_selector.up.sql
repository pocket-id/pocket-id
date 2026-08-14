DROP TRIGGER IF EXISTS prevent_agent_identifier_change_with_credentials ON users;
DROP FUNCTION IF EXISTS enforce_authentication_path_transition();
DROP TRIGGER IF EXISTS prevent_passkey_on_runtime_path ON webauthn_credentials;
DROP FUNCTION IF EXISTS enforce_passkey_path();
DROP TRIGGER IF EXISTS prevent_runtime_credential_on_passkey_path ON runtime_credentials;
DROP FUNCTION IF EXISTS enforce_runtime_credential_path();
DROP INDEX IF EXISTS idx_users_agent_identifier;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_agent_identifier;

ALTER TABLE users ADD COLUMN is_agent BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE users SET is_agent = TRUE WHERE agent_identifier IS NOT NULL;
ALTER TABLE users DROP COLUMN agent_identifier;

CREATE FUNCTION enforce_authentication_path_transition() RETURNS trigger AS $$
BEGIN
    IF OLD.is_agent IS DISTINCT FROM NEW.is_agent AND (
        EXISTS (SELECT 1 FROM webauthn_credentials WHERE user_id = OLD.id)
        OR EXISTS (SELECT 1 FROM runtime_credentials WHERE user_id = OLD.id AND revoked_at IS NULL)
    ) THEN
        RAISE EXCEPTION 'authentication path cannot change while credentials exist';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_is_agent_change_with_credentials
BEFORE UPDATE OF is_agent ON users
FOR EACH ROW EXECUTE FUNCTION enforce_authentication_path_transition();

CREATE FUNCTION enforce_passkey_path() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND is_agent) THEN
        RAISE EXCEPTION 'passkeys are not allowed on the runtime authentication path';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_passkey_on_runtime_path
BEFORE INSERT ON webauthn_credentials
FOR EACH ROW EXECUTE FUNCTION enforce_passkey_path();

CREATE FUNCTION enforce_runtime_credential_path() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND NOT is_agent) THEN
        RAISE EXCEPTION 'runtime credentials require the runtime authentication path';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_runtime_credential_on_passkey_path
BEFORE INSERT ON runtime_credentials
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_credential_path();

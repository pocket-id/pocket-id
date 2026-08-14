-- FCA03 creates runtime proof persistence and enforces credential-path exclusivity at the database boundary
ALTER TABLE users ADD COLUMN agent_identifier TEXT;

ALTER TABLE users ADD CONSTRAINT chk_users_agent_identifier
    CHECK (agent_identifier IS NULL OR agent_identifier ~ '^[a-z0-9]([a-z0-9._-]*[a-z0-9])?$');

CREATE UNIQUE INDEX idx_users_agent_identifier ON users (agent_identifier);

CREATE TABLE runtime_credentials (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    name TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    public_key BYTEA NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_runtime_credentials_algorithm CHECK (algorithm = 'Ed25519'),
    CONSTRAINT chk_runtime_credentials_public_key CHECK (octet_length(public_key) = 32)
);

CREATE INDEX idx_runtime_credentials_user_id ON runtime_credentials (user_id);
CREATE INDEX idx_runtime_credentials_expires_at ON runtime_credentials (expires_at);
CREATE UNIQUE INDEX idx_runtime_credentials_active_user ON runtime_credentials (user_id) WHERE revoked_at IS NULL;

CREATE TABLE runtime_credential_challenges (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    operation TEXT NOT NULL,
    challenge BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    runtime_credential_id UUID REFERENCES runtime_credentials(id) ON DELETE CASCADE,
    credential_name TEXT,
    algorithm TEXT,
    public_key BYTEA,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_runtime_credential_challenges_operation CHECK (operation IN ('register', 'login', 'reauthenticate'))
);

CREATE INDEX idx_runtime_credential_challenges_expires_at ON runtime_credential_challenges (expires_at);

CREATE FUNCTION enforce_authentication_path_transition() RETURNS trigger AS $$
BEGIN
    IF OLD.agent_identifier IS DISTINCT FROM NEW.agent_identifier AND (
        EXISTS (SELECT 1 FROM webauthn_credentials WHERE user_id = OLD.id)
        OR EXISTS (SELECT 1 FROM runtime_credentials WHERE user_id = OLD.id AND revoked_at IS NULL)
    ) THEN
        RAISE EXCEPTION 'authentication path cannot change while credentials exist';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_agent_identifier_change_with_credentials
BEFORE UPDATE OF agent_identifier ON users
FOR EACH ROW EXECUTE FUNCTION enforce_authentication_path_transition();

CREATE FUNCTION enforce_passkey_path() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND agent_identifier IS NOT NULL) THEN
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
    IF EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id AND agent_identifier IS NULL) THEN
        RAISE EXCEPTION 'runtime credentials require the runtime authentication path';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_runtime_credential_on_passkey_path
BEFORE INSERT ON runtime_credentials
FOR EACH ROW EXECUTE FUNCTION enforce_runtime_credential_path();

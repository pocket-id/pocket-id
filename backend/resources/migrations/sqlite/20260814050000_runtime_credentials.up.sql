PRAGMA foreign_keys = OFF;
BEGIN;

ALTER TABLE users ADD COLUMN agent_identifier TEXT;

CREATE UNIQUE INDEX idx_users_agent_identifier ON users (agent_identifier);

CREATE TABLE runtime_credentials (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    name TEXT NOT NULL,
    algorithm TEXT NOT NULL,
    public_key BLOB NOT NULL,
    last_used_at DATETIME,
    expires_at DATETIME,
    revoked_at DATETIME,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_runtime_credentials_algorithm CHECK (algorithm = 'Ed25519'),
    CONSTRAINT chk_runtime_credentials_public_key CHECK (length(public_key) = 32)
);

CREATE INDEX idx_runtime_credentials_user_id ON runtime_credentials (user_id);
CREATE INDEX idx_runtime_credentials_expires_at ON runtime_credentials (expires_at);
CREATE UNIQUE INDEX idx_runtime_credentials_active_user ON runtime_credentials (user_id) WHERE revoked_at IS NULL;

CREATE TABLE runtime_credential_challenges (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    operation TEXT NOT NULL,
    challenge BLOB NOT NULL,
    expires_at DATETIME NOT NULL,
    runtime_credential_id TEXT REFERENCES runtime_credentials(id) ON DELETE CASCADE,
    credential_name TEXT,
    algorithm TEXT,
    public_key BLOB,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_runtime_credential_challenges_operation CHECK (operation IN ('register', 'login', 'reauthenticate'))
);

CREATE INDEX idx_runtime_credential_challenges_expires_at ON runtime_credential_challenges (expires_at);

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

ALTER TABLE webauthn_sessions ADD COLUMN extensions JSONB NOT NULL DEFAULT '{}';

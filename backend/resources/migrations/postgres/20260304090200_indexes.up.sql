CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions (expires_at);
CREATE INDEX IF NOT EXISTS idx_one_time_access_tokens_expires_at ON one_time_access_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_oidc_authorization_codes_expires_at ON oidc_authorization_codes (expires_at);
CREATE INDEX IF NOT EXISTS idx_oidc_refresh_tokens_expires_at ON oidc_refresh_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_reauthentication_tokens_expires_at ON reauthentication_tokens (expires_at);
CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_expires_at ON email_verification_tokens (expires_at);
ALTER TABLE oidc_clients ADD COLUMN access_token_duration_seconds INTEGER NOT NULL DEFAULT 3600;
ALTER TABLE oidc_clients ADD COLUMN refresh_token_duration_seconds INTEGER NOT NULL DEFAULT 2592000;

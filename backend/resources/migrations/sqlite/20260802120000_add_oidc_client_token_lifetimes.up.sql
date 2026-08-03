ALTER TABLE oidc_clients ADD COLUMN access_token_duration_minutes INTEGER NOT NULL DEFAULT 60;
ALTER TABLE oidc_clients ADD COLUMN refresh_token_duration_minutes INTEGER NOT NULL DEFAULT 43200;

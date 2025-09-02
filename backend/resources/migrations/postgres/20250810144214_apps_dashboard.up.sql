ALTER TABLE oidc_clients ADD COLUMN launch_url TEXT;

ALTER TABLE user_authorized_oidc_clients ADD COLUMN last_used_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp;
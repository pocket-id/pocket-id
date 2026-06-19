ALTER TABLE oidc_device_codes ADD COLUMN interval_seconds INTEGER NOT NULL DEFAULT 5;
ALTER TABLE oidc_device_codes ADD COLUMN last_polled_at DATETIME;

PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_authorization_codes
    ADD COLUMN authentication_method TEXT NOT NULL DEFAULT '';
ALTER TABLE oidc_refresh_tokens
    ADD COLUMN authentication_method TEXT NOT NULL DEFAULT '';
ALTER TABLE oidc_device_codes
    ADD COLUMN authentication_method TEXT NOT NULL DEFAULT '';

COMMIT;
PRAGMA foreign_keys= ON;

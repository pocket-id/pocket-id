ALTER TABLE oidc_authorization_codes DROP COLUMN authentication_method;
ALTER TABLE oidc_refresh_tokens DROP COLUMN authentication_method;
ALTER TABLE oidc_device_codes DROP COLUMN authentication_method;

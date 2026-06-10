DROP TABLE IF EXISTS oidc_pushed_authorization_requests;

ALTER TABLE oidc_clients DROP COLUMN requires_pushed_authorization_requests;

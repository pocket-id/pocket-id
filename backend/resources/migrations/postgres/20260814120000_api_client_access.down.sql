DROP TABLE IF EXISTS oidc_clients_allowed_apis;
ALTER TABLE api_permissions DROP COLUMN allowed_for_cimd_clients;
ALTER TABLE apis DROP COLUMN allow_cimd_clients;

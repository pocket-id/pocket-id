PRAGMA foreign_keys=OFF;
BEGIN;

DROP TABLE IF EXISTS oidc_clients_allowed_api_permissions;
DROP TABLE IF EXISTS api_permissions;
DROP TABLE IF EXISTS apis;

COMMIT;
PRAGMA foreign_keys=ON;

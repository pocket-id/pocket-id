PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX IF EXISTS idx_oidc_refresh_tokens_token;
DROP TABLE IF EXISTS oidc_refresh_tokens;
COMMIT;
PRAGMA foreign_keys=ON;

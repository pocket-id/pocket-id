PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX IF EXISTS idx_api_keys_key;
DROP TABLE IF EXISTS api_keys;
COMMIT;
PRAGMA foreign_keys=ON;

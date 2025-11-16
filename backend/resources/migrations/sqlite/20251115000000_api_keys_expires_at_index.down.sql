PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX idx_api_keys_expires_at;
COMMIT;
PRAGMA foreign_keys=ON;

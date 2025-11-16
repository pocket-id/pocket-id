PRAGMA foreign_keys=OFF;
BEGIN;
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);
COMMIT;
PRAGMA foreign_keys=ON;

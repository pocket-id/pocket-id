PRAGMA foreign_keys=OFF;
BEGIN;

-- One-time access tokens are now stored in the actor state store
DROP TABLE IF EXISTS one_time_access_tokens;

COMMIT;
PRAGMA foreign_keys=ON;

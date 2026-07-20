PRAGMA foreign_keys=OFF;
BEGIN;

-- One-time access tokens are now stored in the actor state store (each token is its own actor,
-- keyed by the token value, with a TTL matching its expiration), so the table is no longer used.
DROP TABLE IF EXISTS one_time_access_tokens;

COMMIT;
PRAGMA foreign_keys=ON;

PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX IF EXISTS idx_signup_tokens_expires_at;
DROP INDEX IF EXISTS idx_signup_tokens_token;
DROP TABLE IF EXISTS signup_tokens;
COMMIT;
PRAGMA foreign_keys=ON;

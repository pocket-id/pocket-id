PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE webauthn_sessions DROP COLUMN credential_params;
COMMIT;
PRAGMA foreign_keys=ON;

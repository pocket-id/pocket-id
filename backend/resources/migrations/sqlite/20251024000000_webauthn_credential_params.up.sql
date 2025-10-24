PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE webauthn_sessions ADD COLUMN credential_params TEXT NOT NULL DEFAULT '[]';
COMMIT;
PRAGMA foreign_keys=ON;

PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE webauthn_credentials ADD COLUMN backup_eligible BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN backup_state BOOLEAN NOT NULL DEFAULT FALSE;
COMMIT;
PRAGMA foreign_keys=ON;

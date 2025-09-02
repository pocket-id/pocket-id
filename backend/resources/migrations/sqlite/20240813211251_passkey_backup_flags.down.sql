PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE webauthn_credentials DROP COLUMN backup_eligible;
ALTER TABLE webauthn_credentials DROP COLUMN backup_state;
COMMIT;
PRAGMA foreign_keys=ON;

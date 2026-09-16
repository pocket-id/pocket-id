PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE webauthn_credentials DROP COLUMN icon_hidden;
ALTER TABLE webauthn_credentials DROP COLUMN aaguid;
COMMIT;
PRAGMA foreign_keys=ON;

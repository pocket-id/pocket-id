PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients
	ADD COLUMN pkce_supported BOOLEAN NOT NULL DEFAULT 0;

COMMIT;
PRAGMA foreign_keys= ON;
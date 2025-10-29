PRAGMA foreign_keys = OFF;
BEGIN;

ALTER TABLE users ADD COLUMN has_custom_profile_picture NUMERIC NOT NULL DEFAULT FALSE;

COMMIT;
PRAGMA foreign_keys = ON;
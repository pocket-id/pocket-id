PRAGMA foreign_keys = OFF;
BEGIN;

ALTER TABLE users DROP COLUMN has_custom_profile_picture;

COMMIT;
PRAGMA foreign_keys = ON;
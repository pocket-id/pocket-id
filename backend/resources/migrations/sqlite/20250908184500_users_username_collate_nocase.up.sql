PRAGMA foreign_keys=OFF;
BEGIN;

/* Rebuild users table to apply COLLATE NOCASE to username for ASCII-only case-insensitive comparisons */
CREATE TABLE users_new
(
    id         TEXT                  NOT NULL PRIMARY KEY,
    created_at DATETIME,
    username   TEXT                  NOT NULL UNIQUE COLLATE NOCASE,
    email      TEXT                  NOT NULL UNIQUE,
    first_name TEXT,
    last_name  TEXT,
    is_admin   NUMERIC DEFAULT FALSE NOT NULL,
    locale     TEXT,
    ldap_id    TEXT,
    disabled   NUMERIC DEFAULT FALSE NOT NULL
);

/* Copy existing data, normalizing username to lowercase to avoid conflicts */
INSERT INTO users_new (id, created_at, username, email, first_name, last_name, is_admin, locale, ldap_id, disabled)
SELECT id,
       created_at,
       lower(username),
       email,
       first_name,
       last_name,
       is_admin,
       locale,
       ldap_id,
       disabled
FROM users;

/* Replace old table */
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

/* Drop redundant functional unique index if previously created */
DROP INDEX IF EXISTS users_username_lower_unique;

COMMIT;
PRAGMA foreign_keys=ON;
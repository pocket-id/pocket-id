PRAGMA foreign_keys = OFF;

BEGIN;

-- 1. Create the old table definition with NUMERIC columns
CREATE TABLE users_old
(
    id           TEXT                  NOT NULL
        PRIMARY KEY,
    created_at   DATETIME,
    username     TEXT COLLATE NOCASE   NOT NULL
        UNIQUE,
    email        TEXT                  NOT NULL
        UNIQUE,
    first_name   TEXT,
    last_name    TEXT                  NOT NULL,
    display_name TEXT                  NOT NULL,
    is_admin     NUMERIC DEFAULT FALSE NOT NULL,
    ldap_id      TEXT,
    locale       TEXT,
    disabled     NUMERIC DEFAULT FALSE NOT NULL
);

-- 2. Copy all data back, converting booleans to numeric (0/1)
INSERT INTO users_old (
    id,
    created_at,
    username,
    email,
    first_name,
    last_name,
    display_name,
    is_admin,
    ldap_id,
    locale,
    disabled
)
SELECT
    id,
    created_at,
    username,
    email,
    first_name,
    last_name,
    display_name,
    CASE WHEN is_admin THEN 1 ELSE 0 END,
    ldap_id,
    locale,
    CASE WHEN disabled THEN 1 ELSE 0 END
FROM users;

-- 3. Drop the current table
DROP TABLE users;

-- 4. Rename the old-style table back to the original name
ALTER TABLE users_old RENAME TO users;

-- 5. Recreate index
CREATE UNIQUE INDEX users_ldap_id
    ON users (ldap_id);

COMMIT;

PRAGMA foreign_keys = ON;
PRAGMA foreign_keys = OFF;

BEGIN;

-- 1. Create a new table with BOOLEAN columns
CREATE TABLE users_new
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
    is_admin     BOOLEAN DEFAULT FALSE NOT NULL,
    ldap_id      TEXT,
    locale       TEXT,
    disabled     BOOLEAN DEFAULT FALSE NOT NULL
);

-- 2. Copy all existing data, converting numeric bools to real booleans
INSERT INTO users_new (
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
    CASE WHEN is_admin != 0 THEN TRUE ELSE FALSE END,
    ldap_id,
    locale,
    CASE WHEN disabled != 0 THEN TRUE ELSE FALSE END
FROM users;

-- 3. Drop old table
DROP TABLE users;

-- 4. Rename new table to original name
ALTER TABLE users_new RENAME TO users;

-- 5. Recreate index
CREATE UNIQUE INDEX users_ldap_id
    ON users (ldap_id);

COMMIT;

PRAGMA foreign_keys = ON;
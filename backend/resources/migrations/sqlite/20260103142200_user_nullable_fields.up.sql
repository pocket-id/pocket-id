PRAGMA foreign_keys= OFF;
BEGIN;

CREATE TABLE users_new
(
    id           TEXT                  NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    updated_at   DATETIME,
    username     TEXT COLLATE NOCASE   NOT NULL UNIQUE,
    email        TEXT UNIQUE,
    first_name   TEXT                  NOT NULL,
    last_name    TEXT                  NOT NULL,
    display_name TEXT                  NOT NULL,
    is_admin     BOOLEAN DEFAULT FALSE NOT NULL,
    ldap_id      TEXT UNIQUE,
    locale       TEXT,
    disabled     BOOLEAN DEFAULT FALSE NOT NULL
);

INSERT INTO users_new (
    id,
    created_at,
    updated_at,
    username,
    email,
    first_name,
    last_name,
    display_name,
    is_admin,
    ldap_id,
    locale,
    disabled
) SELECT
    id,
    created_at,
    updated_at,
    username,
    email,
    first_name,
    last_name,
    display_name,
    is_admin,
    ldap_id,
    locale,
    disabled FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;


COMMIT;
PRAGMA foreign_keys= ON;

PRAGMA foreign_keys = OFF;
BEGIN;

CREATE TABLE users_new
(
    id           TEXT    NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    username     TEXT    NOT NULL COLLATE NOCASE UNIQUE,
    email        TEXT    UNIQUE,
    first_name   TEXT,
    last_name    TEXT    NOT NULL,
    display_name TEXT    NOT NULL,
    is_admin     NUMERIC NOT NULL DEFAULT FALSE,
    ldap_id      TEXT,
    locale       TEXT,
    disabled     NUMERIC NOT NULL DEFAULT FALSE
);

INSERT INTO users_new (id, created_at, username, email, first_name, last_name, display_name, is_admin, ldap_id, locale,
                       disabled)
SELECT id,
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
FROM users;

DROP TABLE users;

ALTER TABLE users_new
    RENAME TO users;

COMMIT;
PRAGMA foreign_keys = ON;
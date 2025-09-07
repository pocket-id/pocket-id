PRAGMA foreign_keys= OFF;
BEGIN;

CREATE TABLE users_new
(
    id           TEXT                  not null
        primary key,
    created_at   DATETIME,
    username     TEXT                  not null
        unique,
    email        TEXT                  not null
        unique,
    first_name   TEXT,
                     last_name TEXT,
    display_name TEXT                  not null,
    is_admin     NUMERIC default FALSE not null,
    ldap_id      TEXT,
    locale       TEXT,
    disabled     NUMERIC default FALSE not null
);

INSERT INTO users_new (id, created_at, username, email, first_name, last_name, display_name, is_admin, ldap_id,
                            locale, disabled)
SELECT id,
       created_at,
       username,
       email,
       first_name,
       last_name,
       trim(coalesce(first_name, '') || ' ' || coalesce(last_name, '')),
       is_admin,
       ldap_id,
       locale,
       disabled
FROM users;

DROP TABLE users;
ALTER TABLE users_new
    RENAME TO users;

CREATE UNIQUE INDEX users_ldap_id ON users (ldap_id);

COMMIT;
PRAGMA foreign_keys= ON;
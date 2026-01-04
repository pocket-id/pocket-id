-- This migration is part of v2

PRAGMA foreign_keys = OFF;

BEGIN;

-- 1. Create a new table with BOOLEAN columns
CREATE TABLE users_new
(
    id           TEXT                  NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    username     TEXT COLLATE NOCASE   NOT NULL UNIQUE,
    email        TEXT UNIQUE,
    first_name   TEXT                  NOT NULL,
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
CREATE UNIQUE INDEX users_ldap_id ON users (ldap_id);

-- 6. Create temporary table with changed credential_id type to BLOB
CREATE TABLE webauthn_credentials_dg_tmp
(
    id               TEXT PRIMARY KEY,
    created_at       DATETIME             NOT NULL,
    name             TEXT                 NOT NULL,
    credential_id    BLOB                 NOT NULL UNIQUE,
    public_key       BLOB                 NOT NULL,
    attestation_type TEXT                 NOT NULL,
    transport        BLOB                 NOT NULL,
    user_id          TEXT REFERENCES users ON DELETE CASCADE,
    backup_eligible  BOOLEAN DEFAULT FALSE NOT NULL,
    backup_state     BOOLEAN DEFAULT FALSE NOT NULL
);

-- 7. Copy existing data into the temporary table
INSERT INTO webauthn_credentials_dg_tmp (
    id,
    created_at,
    name,
    credential_id,
    public_key,
    attestation_type,
    transport,
    user_id,
    backup_eligible,
    backup_state
)
SELECT
    id,
    created_at,
    name,
    credential_id,
    public_key,
    attestation_type,
    transport,
    user_id,
    backup_eligible,
    backup_state
FROM webauthn_credentials;

-- 8. Drop old table
DROP TABLE webauthn_credentials;

-- 9. Rename temporary table to original name
ALTER TABLE webauthn_credentials_dg_tmp
    RENAME TO webauthn_credentials;

-- 10. Create temporary table with credential_params type changed to BLOB
CREATE TABLE webauthn_sessions_dg_tmp
(
    id                TEXT              NOT NULL PRIMARY KEY,
    created_at        DATETIME,
    challenge         TEXT              NOT NULL UNIQUE,
    expires_at        DATETIME          NOT NULL,
    user_verification TEXT              NOT NULL,
    credential_params BLOB DEFAULT '[]' NOT NULL
);

-- 11. Copy existing data into the temporary sessions table
INSERT INTO webauthn_sessions_dg_tmp (
    id,
    created_at,
    challenge,
    expires_at,
    user_verification,
    credential_params
)
SELECT
    id,
    created_at,
    challenge,
    expires_at,
    user_verification,
    credential_params
FROM webauthn_sessions;

-- 12. Drop old table
DROP TABLE webauthn_sessions;

-- 13. Rename temporary sessions table to original name
ALTER TABLE webauthn_sessions_dg_tmp
    RENAME TO webauthn_sessions;

COMMIT;

PRAGMA foreign_keys = ON;
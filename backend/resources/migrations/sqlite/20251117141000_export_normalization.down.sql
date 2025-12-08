PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE users_old
(
    id           TEXT                   NOT NULL PRIMARY KEY,
    created_at   DATETIME,
    username     TEXT COLLATE NOCASE    NOT NULL UNIQUE,
    email        TEXT                   NOT NULL UNIQUE,
    first_name   TEXT,
    last_name    TEXT                   NOT NULL,
    display_name TEXT                   NOT NULL,
    is_admin     NUMERIC DEFAULT 0      NOT NULL,
    ldap_id      TEXT,
    locale       TEXT,
    disabled     NUMERIC DEFAULT 0      NOT NULL
);

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

DROP TABLE users;

ALTER TABLE users_old RENAME TO users;

CREATE UNIQUE INDEX users_ldap_id ON users (ldap_id);



CREATE TABLE webauthn_credentials_old
(
    id               TEXT                   PRIMARY KEY,
    created_at       DATETIME               NOT NULL,
    name             TEXT                   NOT NULL,
    credential_id    TEXT                   NOT NULL UNIQUE,
    public_key       BLOB                   NOT NULL,
    attestation_type TEXT                   NOT NULL,
    transport        BLOB                   NOT NULL,
    user_id          TEXT REFERENCES users ON DELETE CASCADE,
    backup_eligible  NUMERIC DEFAULT 0      NOT NULL,
    backup_state     NUMERIC DEFAULT 0      NOT NULL
);

INSERT INTO webauthn_credentials_old (
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
    CASE WHEN backup_eligible THEN 1 ELSE 0 END,
    CASE WHEN backup_state THEN 1 ELSE 0 END
FROM webauthn_credentials;

DROP TABLE webauthn_credentials;

ALTER TABLE webauthn_credentials_old RENAME TO webauthn_credentials;



CREATE TABLE webauthn_sessions_old
(
    id                TEXT              NOT NULL PRIMARY KEY,
    created_at        DATETIME,
    challenge         TEXT              NOT NULL UNIQUE,
    expires_at        DATETIME          NOT NULL,
    user_verification TEXT              NOT NULL,
    credential_params TEXT DEFAULT '[]' NOT NULL
);

INSERT INTO webauthn_sessions_old (
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

DROP TABLE webauthn_sessions;

ALTER TABLE webauthn_sessions_old RENAME TO webauthn_sessions;

COMMIT;

PRAGMA foreign_keys = ON;
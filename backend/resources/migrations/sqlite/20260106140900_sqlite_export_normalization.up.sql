PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE oidc_clients_dg_tmp
(
    id                        TEXT PRIMARY KEY,
    created_at                DATETIME NOT NULL,
    name                      TEXT,
    secret                    TEXT,
    callback_urls             BLOB,
    image_type                TEXT,
    created_by_id             TEXT REFERENCES users ON DELETE SET NULL,
    is_public                 BOOLEAN DEFAULT FALSE,
    pkce_enabled              BOOLEAN DEFAULT FALSE,
    logout_callback_urls      BLOB,
    credentials               BLOB,
    launch_url                TEXT,
    requires_reauthentication BOOLEAN NOT NULL DEFAULT FALSE,
    dark_image_type           TEXT,
    is_group_restricted       BOOLEAN NOT NULL DEFAULT 0
);

INSERT INTO oidc_clients_dg_tmp (
    id, created_at, name, secret, callback_urls, image_type, created_by_id,
    is_public, pkce_enabled, logout_callback_urls, credentials, launch_url,
    requires_reauthentication, dark_image_type, is_group_restricted
)
SELECT
    id,
    created_at,
    name,
    secret,
    callback_urls,
    image_type,
    created_by_id,
    is_public,
    pkce_enabled,
    logout_callback_urls,
    credentials,
    launch_url,
    requires_reauthentication,
    dark_image_type,
    is_group_restricted
FROM oidc_clients;

DROP TABLE oidc_clients;

ALTER TABLE oidc_clients_dg_tmp RENAME TO oidc_clients;

COMMIT;
PRAGMA foreign_keys=ON;

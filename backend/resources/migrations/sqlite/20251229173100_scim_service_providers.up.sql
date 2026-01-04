PRAGMA foreign_keys= OFF;
BEGIN;

CREATE TABLE scim_service_providers
(
    id             TEXT PRIMARY KEY,
    created_at     DATETIME NOT NULL,
    endpoint       TEXT     NOT NULL,
    token          TEXT     NOT NULL,
    last_synced_at DATETIME,
    oidc_client_id TEXT     NOT NULL,
    FOREIGN KEY (oidc_client_id) REFERENCES oidc_clients (id) ON DELETE CASCADE
);

ALTER TABLE users
    ADD COLUMN updated_at DATETIME;

ALTER TABLE user_groups
    ADD COLUMN updated_at DATETIME;

COMMIT;
PRAGMA foreign_keys= ON;

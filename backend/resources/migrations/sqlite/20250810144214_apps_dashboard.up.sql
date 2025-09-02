PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE oidc_clients ADD COLUMN launch_url TEXT;

CREATE TABLE user_authorized_oidc_clients_new
(
    scope     TEXT,
    user_id   TEXT,
    client_id TEXT REFERENCES oidc_clients,
    last_used_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, client_id)
);

INSERT INTO user_authorized_oidc_clients_new (scope, user_id, client_id, last_used_at)
SELECT scope, user_id, client_id, unixepoch() FROM user_authorized_oidc_clients;

DROP TABLE user_authorized_oidc_clients;
ALTER TABLE user_authorized_oidc_clients_new RENAME TO user_authorized_oidc_clients;

COMMIT;
PRAGMA foreign_keys=ON;

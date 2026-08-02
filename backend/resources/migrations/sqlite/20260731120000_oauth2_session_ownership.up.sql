PRAGMA foreign_keys = OFF;
BEGIN;

CREATE TABLE oauth2_sessions_new (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    request_id TEXT NOT NULL,
    client_id TEXT NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    access_token_signature TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    request_data BLOB NOT NULL,
    expires_at DATETIME,
    CONSTRAINT chk_oauth2_sessions_client_id
        CHECK (client_id = json_extract(CAST(request_data AS TEXT), '$.client_id'))
);

INSERT INTO oauth2_sessions_new (
    id,
    created_at,
    kind,
    key,
    request_id,
    client_id,
    access_token_signature,
    active,
    request_data,
    expires_at
)
SELECT
    id,
    created_at,
    kind,
    key,
    request_id,
    json_extract(CAST(request_data AS TEXT), '$.client_id'),
    access_token_signature,
    active,
    request_data,
    expires_at
FROM oauth2_sessions
WHERE json_valid(CAST(request_data AS TEXT))
  AND EXISTS (
      SELECT 1
      FROM oidc_clients
      WHERE oidc_clients.id = json_extract(CAST(oauth2_sessions.request_data AS TEXT), '$.client_id')
  );

DROP TABLE oauth2_sessions;
ALTER TABLE oauth2_sessions_new RENAME TO oauth2_sessions;

CREATE UNIQUE INDEX idx_oauth2_sessions_kind_key ON oauth2_sessions (kind, key);
CREATE INDEX idx_oauth2_sessions_kind_request ON oauth2_sessions (kind, request_id);
CREATE INDEX idx_oauth2_sessions_expires_at ON oauth2_sessions (expires_at);
CREATE INDEX idx_oauth2_sessions_client_subject
    ON oauth2_sessions (client_id, json_extract(CAST(request_data AS TEXT), '$.session.subject'), kind, active);

COMMIT;
PRAGMA foreign_keys = ON;

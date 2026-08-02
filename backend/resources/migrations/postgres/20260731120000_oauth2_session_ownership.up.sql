ALTER TABLE oauth2_sessions
    ADD COLUMN client_id TEXT;

UPDATE oauth2_sessions
SET client_id = request_data ->> 'client_id';

DELETE FROM oauth2_sessions
WHERE client_id IS NULL
   OR NOT EXISTS (
       SELECT 1
       FROM oidc_clients
       WHERE oidc_clients.id = oauth2_sessions.client_id
   );

ALTER TABLE oauth2_sessions
    ALTER COLUMN client_id SET NOT NULL,
    ADD CONSTRAINT fk_oauth2_sessions_client_id
        FOREIGN KEY (client_id) REFERENCES oidc_clients(id) ON DELETE CASCADE,
    ADD CONSTRAINT chk_oauth2_sessions_client_id
        CHECK (client_id = request_data ->> 'client_id');

CREATE INDEX idx_oauth2_sessions_client_subject
    ON oauth2_sessions (client_id, (request_data #>> '{session,subject}'), kind, active);

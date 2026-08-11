PRAGMA foreign_keys= OFF;
BEGIN;

ALTER TABLE oidc_clients ADD COLUMN secret TEXT;

-- Only secrets that were migrated up from this same column can be restored, since the old column stores a bcrypt hash
-- Secrets created while multiple secrets were supported are hashed with SHA-256 and are dropped here
UPDATE oidc_clients
SET secret = (
        SELECT json_extract(value, '$.hash')
        FROM json_each(json_extract(credentials, '$.secrets'))
        WHERE json_extract(value, '$.alg') = 'bcrypt'
        LIMIT 1
)
WHERE credentials IS NOT NULL
  AND credentials != ''
  AND json_valid(credentials)
  AND json_type(credentials, '$.secrets') = 'array';

UPDATE oidc_clients
SET credentials = json_remove(credentials, '$.secrets')
WHERE credentials IS NOT NULL
  AND credentials != ''
  AND json_valid(credentials)
  AND json_type(credentials, '$.secrets') = 'array';

COMMIT;
PRAGMA foreign_keys= ON;

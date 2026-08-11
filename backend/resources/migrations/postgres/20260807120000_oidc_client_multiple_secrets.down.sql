ALTER TABLE oidc_clients ADD COLUMN secret TEXT;

-- Only secrets that were migrated up from this same column can be restored, since the old column stores a bcrypt hash
-- Secrets created while multiple secrets were supported are hashed with SHA-256 and are dropped here
UPDATE oidc_clients
SET secret = (
        SELECT secret_entry ->> 'hash'
        FROM jsonb_array_elements(credentials -> 'secrets') AS secret_entry
        WHERE secret_entry ->> 'alg' = 'bcrypt'
        LIMIT 1
)
WHERE jsonb_typeof(credentials -> 'secrets') = 'array';

UPDATE oidc_clients
SET credentials = credentials - 'secrets'
WHERE jsonb_typeof(credentials -> 'secrets') = 'array';

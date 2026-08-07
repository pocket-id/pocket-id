PRAGMA foreign_keys= OFF;
BEGIN;

-- Move the single client secret into the credentials document, as the only entry of the new "secrets" array
-- Migrated secrets keep their bcrypt hash because the secret's value is not recoverable, and they never expire so that existing integrations keep working
UPDATE oidc_clients
SET credentials = json_set(
        COALESCE(NULLIF(credentials, ''), '{}'),
        '$.secrets',
        json_array(json_object(
                'id', lower(
                        hex(randomblob(4)) || '-' ||
                        hex(randomblob(2)) || '-4' ||
                        substr(hex(randomblob(2)), 2) || '-' ||
                        substr('89ab', abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)), 2) || '-' ||
                        hex(randomblob(6))
                ),
                'alg', 'bcrypt',
                'hash', secret,
                'createdAt', strftime('%Y-%m-%dT%H:%M:%SZ', COALESCE(created_at, strftime('%s', 'now')), 'unixepoch')
        ))
)
WHERE secret IS NOT NULL
  AND secret != ''
  AND (credentials IS NULL OR credentials = '' OR json_valid(credentials));

ALTER TABLE oidc_clients DROP COLUMN secret;

COMMIT;
PRAGMA foreign_keys= ON;

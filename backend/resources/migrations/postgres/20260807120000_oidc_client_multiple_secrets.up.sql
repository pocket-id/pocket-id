-- Move the single client secret into the credentials document, as the only entry of the new "secrets" array
-- Migrated secrets keep their bcrypt hash because the secret's value is not recoverable, and they never expire so that existing integrations keep working
UPDATE oidc_clients
SET credentials = COALESCE(credentials, '{}'::jsonb) || jsonb_build_object(
        'secrets', jsonb_build_array(jsonb_build_object(
                'id', gen_random_uuid()::text,
                'alg', 'bcrypt',
                'hash', secret,
                'createdAt', to_char(COALESCE(created_at, now()) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
        ))
)
WHERE secret IS NOT NULL AND secret <> '';

ALTER TABLE oidc_clients DROP COLUMN secret;

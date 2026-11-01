ALTER TABLE api_keys ADD COLUMN scopes JSONB NOT NULL DEFAULT '[]';

-- Existing keys keep the access they had, which was every scope an API key could hold for their owner's role
UPDATE api_keys SET scopes = CASE
    WHEN (SELECT is_admin FROM users WHERE users.id = api_keys.user_id) THEN '["account:api-keys","account:apps","account:audit-logs","account:passkeys","account:read","account:write","apis:read","apis:write","audit-logs:read","config:read","config:write","groups:read","groups:write","oidc-claim-mapping-policies:read","oidc-claim-mapping-policies:write","oidc-clients:read","oidc-clients:write","users:read","users:write"]'::jsonb
    ELSE '["account:api-keys","account:apps","account:audit-logs","account:passkeys","account:read","account:write"]'::jsonb
END;

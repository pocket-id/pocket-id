ALTER TABLE api_keys ADD COLUMN scopes BLOB NOT NULL DEFAULT X'5B5D';

-- Existing keys keep the access they had, which was every scope an API key could hold for their owner's role
UPDATE api_keys SET scopes = CASE
    WHEN (SELECT is_admin FROM users WHERE users.id = api_keys.user_id) THEN CAST('["account:api-keys","account:apps","account:audit-logs","account:passkeys","account:read","account:write","apis:read","apis:write","audit-logs:read","config:read","config:write","groups:read","groups:write","oidc-claim-mapping-policies:read","oidc-claim-mapping-policies:write","oidc-clients:read","oidc-clients:write","users:read","users:write"]' AS BLOB)
    ELSE CAST('["account:api-keys","account:apps","account:audit-logs","account:passkeys","account:read","account:write"]' AS BLOB)
END;

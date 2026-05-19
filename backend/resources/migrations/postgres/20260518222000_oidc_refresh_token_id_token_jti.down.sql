DROP INDEX IF EXISTS idx_oidc_refresh_tokens_id_token_jti;

ALTER TABLE oidc_refresh_tokens
    DROP COLUMN id_token_jti;

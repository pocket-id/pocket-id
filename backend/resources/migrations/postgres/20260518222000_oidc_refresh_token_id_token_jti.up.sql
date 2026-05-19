ALTER TABLE oidc_refresh_tokens
    ADD COLUMN id_token_jti UUID;

CREATE INDEX idx_oidc_refresh_tokens_id_token_jti
    ON oidc_refresh_tokens(user_id, client_id, id_token_jti);

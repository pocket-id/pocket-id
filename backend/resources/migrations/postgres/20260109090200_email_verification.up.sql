CREATE TABLE email_verification_tokens
(
    id         UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    token      TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    user_id    TEXT        NOT NULL,
    CONSTRAINT email_verification_tokens_user_id_fkey
        FOREIGN KEY (user_id)
            REFERENCES users (id)
            ON DELETE CASCADE
);

ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET email_verified = EXISTS (SELECT 1
                             FROM app_config_variables
                             WHERE key = 'emailsVerified'
                               AND value = 'true');
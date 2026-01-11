CREATE TABLE email_verification_tokens
(
    id         UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    token      TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users ON DELETE CASCADE
);

ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET email_verified = EXISTS (SELECT 1
                             FROM app_config_variables
                             WHERE key = 'emailsVerified'
                               AND value = 'true');
CREATE TABLE signup_tokens (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    usage_limit INTEGER NOT NULL DEFAULT 1,
    usage_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_signup_tokens_token ON signup_tokens(token);
CREATE INDEX idx_signup_tokens_expires_at ON signup_tokens(expires_at);
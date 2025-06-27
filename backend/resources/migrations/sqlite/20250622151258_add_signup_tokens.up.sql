CREATE TABLE signup_tokens (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    usage_limit INTEGER NOT NULL DEFAULT 1,
    usage_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_signup_tokens_token ON signup_tokens(token);
CREATE INDEX idx_signup_tokens_expires_at ON signup_tokens(expires_at);
CREATE TABLE recovery_codes
(
    id         UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users ON DELETE CASCADE,
    code_hash  TEXT        NOT NULL UNIQUE,
    used_at    TIMESTAMPTZ
);

CREATE INDEX idx_recovery_codes_user_id ON recovery_codes (user_id);

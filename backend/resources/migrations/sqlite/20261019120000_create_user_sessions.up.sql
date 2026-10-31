PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE user_sessions (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    user_id TEXT NOT NULL REFERENCES users ON DELETE CASCADE
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);
COMMIT;
PRAGMA foreign_keys=ON;

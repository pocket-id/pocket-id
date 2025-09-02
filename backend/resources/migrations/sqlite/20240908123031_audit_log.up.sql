PRAGMA foreign_keys=OFF;
BEGIN;
CREATE TABLE audit_logs
(
    id               TEXT NOT NULL PRIMARY KEY,
    created_at       DATETIME,
    event            TEXT NOT NULL,
    ip_address       TEXT NOT NULL,
    user_agent       TEXT NOT NULL,
    data             BLOB NOT NULL,
    user_id          TEXT REFERENCES users
);
COMMIT;
PRAGMA foreign_keys=ON;

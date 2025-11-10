PRAGMA foreign_keys=OFF;
BEGIN;
-- The "storage" table contains file data stored in the database
CREATE TABLE storage
(
    path       TEXT NOT NULL PRIMARY KEY,
    data       BLOB NOT NULL,
    size       INTEGER NOT NULL,
    mod_time   INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

COMMIT;
PRAGMA foreign_keys=ON;

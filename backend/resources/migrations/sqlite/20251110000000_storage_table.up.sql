PRAGMA foreign_keys=OFF;
BEGIN;
-- The "storage" table contains file data stored in the database
CREATE TABLE storage
(
    path       TEXT NOT NULL PRIMARY KEY,
    data       BLOB NOT NULL,
    size       INTEGER NOT NULL,
    mod_time   DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

COMMIT;
PRAGMA foreign_keys=ON;

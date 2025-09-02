PRAGMA foreign_keys=OFF;
BEGIN;
-- The "kv" tables contains miscellaneous key-value pairs
CREATE TABLE kv
(
    "key"  TEXT NOT NULL PRIMARY KEY,
    "value" TEXT NOT NULL
);

COMMIT;
PRAGMA foreign_keys=ON;

-- The "storage" table contains file data stored in the database
CREATE TABLE storage
(
    path       TEXT NOT NULL PRIMARY KEY,
    data       BYTEA NOT NULL,
    size       BIGINT NOT NULL,
    mod_time   DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);

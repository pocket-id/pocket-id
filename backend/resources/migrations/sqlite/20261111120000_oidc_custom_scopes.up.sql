CREATE TABLE oidc_custom_scopes (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE email_signing_settings (
 id INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
 enabled BOOLEAN NOT NULL DEFAULT FALSE,
 certificate_chain TEXT NOT NULL DEFAULT '',
 private_key TEXT NOT NULL DEFAULT ''
);

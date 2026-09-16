ALTER TABLE webauthn_credentials ADD COLUMN aaguid TEXT NOT NULL DEFAULT '';
ALTER TABLE webauthn_credentials ADD COLUMN icon_hidden BOOLEAN NOT NULL DEFAULT FALSE;

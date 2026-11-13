ALTER TABLE webauthn_credentials ADD COLUMN attestation_verified BOOLEAN NOT NULL DEFAULT FALSE;

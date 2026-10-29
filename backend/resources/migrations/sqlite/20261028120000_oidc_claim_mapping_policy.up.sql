PRAGMA foreign_keys=OFF;
BEGIN;

CREATE TABLE oidc_claim_mapping_policies (
    id TEXT NOT NULL PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    claim_mappings BLOB NOT NULL
);

ALTER TABLE oidc_clients ADD COLUMN claim_mapping_policy_id TEXT REFERENCES oidc_claim_mapping_policies(id);

COMMIT;
PRAGMA foreign_keys=ON;

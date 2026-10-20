CREATE TABLE oidc_claim_mapping_policies (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    claim_mappings JSONB NOT NULL
);

ALTER TABLE oidc_clients ADD COLUMN claim_mapping_policy_id UUID REFERENCES oidc_claim_mapping_policies(id);

CREATE TABLE oidc_claim_mapping_policies (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    name TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    claim_mappings JSONB NOT NULL
);

CREATE UNIQUE INDEX idx_oidc_claim_mapping_policies_default ON oidc_claim_mapping_policies (is_default) WHERE is_default;

ALTER TABLE oidc_clients ADD COLUMN claim_mapping_policy_id UUID REFERENCES oidc_claim_mapping_policies(id) ON DELETE SET NULL;

-- These keys became reserved, so existing custom claims with them would override the claims Pocket ID sets itself
DELETE FROM custom_claims WHERE key IN ('picture', 'at_hash', 'c_hash', 'rat', 'sid', 'scope', 'scp');

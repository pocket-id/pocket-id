PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE oidc_clients DROP COLUMN claim_mapping_policy_id;
DROP TABLE oidc_claim_mapping_policies;

COMMIT;
PRAGMA foreign_keys=ON;

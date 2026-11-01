export type Feature =
	| 'ldap'
	| 'scim'
	| 'customClaims'
	| 'apis'
	| 'apiKeys'
	| 'cimd'
	| 'federatedCredentials'
	| 'claimMappings';

// A feature in use is always enabled, and only an unused one can be turned off
export type Features = Record<Feature, { enabled: boolean; inUse: boolean }>;

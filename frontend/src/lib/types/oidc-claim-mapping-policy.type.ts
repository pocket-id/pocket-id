export type OidcClaimMappingPolicySourceType = 'user_field' | 'custom_claim' | 'static';

// Kept in sync with IsProtocolClaim in backend/internal/common/claims.go
export const RESERVED_CLAIM_NAMES = [
	'sub',
	'iss',
	'aud',
	'exp',
	'iat',
	'nbf',
	'jti',
	'auth_time',
	'nonce',
	'acr',
	'amr',
	'azp',
	'at_hash',
	'c_hash',
	'rat',
	'sid',
	'scope',
	'scp',
	'client_id',
	'type'
];

export const USER_FIELDS = [
	'id',
	'email',
	'email_verified',
	'first_name',
	'last_name',
	'full_name',
	'display_name',
	'username',
	'locale',
	'picture',
	'groups'
] as const;

export type OidcUserField = (typeof USER_FIELDS)[number];

// The built-in scopes a claim can be released with, which custom scopes extend
export const MAPPING_SCOPES = ['openid', 'profile', 'email', 'groups', 'offline_access'];

export type OidcClaimMappingDto = {
	claimName: string;
	sourceType: OidcClaimMappingPolicySourceType;
	sourceValue: string;
	scope: string[];
	accessToken: boolean;
	idToken: boolean;
	userInfo: boolean;
};

// ID of the policy the backend inserts on startup, which is read-only except for its default flag
export const builtInClaimMappingPolicyId = '00000000-0000-0000-0000-000000000000';

export type OidcClaimMappingPolicyMetadataDto = {
	id: string;
	name: string;
	isDefault: boolean;
	createdAt: string;
	claimCount: number;
	// Includes every client without a policy of its own when the policy is the default
	clientCount: number;
};

export type OidcClaimMappingPolicy = OidcClaimMappingPolicyMetadataDto & {
	claimMappings: OidcClaimMappingDto[];
};

export type OidcClaimMappingPolicyInput = {
	name: string;
	isDefault: boolean;
	claimMappings: OidcClaimMappingDto[];
};

export type OidcClaimMappingsPreview = {
	idToken: Record<string, unknown> | null;
	accessToken: Record<string, unknown>;
	userInfo: Record<string, unknown> | null;
};

export type CustomScope = {
	id: string;
	createdAt: string;
	key: string;
	name: string;
	description: string;
	// The names of the policies that release claims with the scope, only set when listing
	policies: string[] | null;
};

export type CustomScopeInput = Omit<CustomScope, 'id' | 'createdAt' | 'policies'>;

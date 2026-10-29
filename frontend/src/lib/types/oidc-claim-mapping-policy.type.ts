export type OidcClaimMappingPolicySourceType = 'user_field' | 'custom_claim' | 'static';

/**
 * Claims the OIDC layer issues itself, which a mapping must not overwrite. Kept in sync with
 * `isReservedClaimsForMapping` in the backend's claimmappingpolicy service.
 */
export const RESERVED_CLAIM_NAMES = [
	'sub',
	'iss',
	'aud',
	'exp',
	'iat',
	'auth_time',
	'nonce',
	'acr',
	'amr',
	'azp',
	'nbf',
	'jti'
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

export const MAPPING_SCOPES = [
	'openid',
	'profile',
	'email',
	'groups',
	'address',
	'phone',
	'offline_access'
] as const;

export type OidcScope = (typeof MAPPING_SCOPES)[number];

export type OidcClaimMappingDto = {
	claimName: string;
	sourceType: OidcClaimMappingPolicySourceType;
	sourceValue: string;
	scope: OidcScope[];
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

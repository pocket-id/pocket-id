export type OidcClaimMappingPolicySourceType = 'user_field' | 'custom_claim' | 'static';

/**
 * Claims the OIDC layer issues itself, which a mapping must not overwrite. Kept in sync with
 * `reservedClaimsForMapping` in the backend's oidc_claim_mapping_policy_service.go.
 */
export const RESERVED_CLAIM_NAMES = [
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
] as const;

export type OidcUserField =
	| 'id'
	| 'email'
	| 'email_verified'
	| 'first_name'
	| 'last_name'
	| 'full_name'
	| 'display_name'
	| 'username'
	| 'locale'
	| 'picture'
	| 'groups';

export type OidcScope =
	| 'openid'
	| 'profile'
	| 'email'
	| 'address'
	| 'phone'
	| 'offline_access'
	| 'groups';

export type OidcClaimMappingDto = {
	claimName: string;
	sourceType: OidcClaimMappingPolicySourceType;
	sourceValue: string;
	scope: OidcScope[];
	accessToken: boolean;
	idToken: boolean;
	userInfo: boolean;
};

export type OidcClaimMappingPolicyMetadataDto = {
	id: string;
	name: string;
	isDefault: boolean;
};

export type OidcClaimMappingPolicyDto = OidcClaimMappingPolicyCreate & {
	id?: string;
};

export type OidcClaimMappingPolicy = OidcClaimMappingPolicyDto;

export type OidcClaimMappingPolicyCreate = {
	name: string;
	isDefault: boolean;
	claimMappings: OidcClaimMappingDto[];
};

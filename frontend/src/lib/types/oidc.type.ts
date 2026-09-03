import type { Jwk } from '$lib/utils/jwk-util';
import type { UserGroup, UserGroupMinimal } from './user-group.type';

export type OidcClientType = 'standard' | 'cimd';

export type OidcClientMetaData = {
	id: string;
	name: string;
	description: string;
	hasLogo: boolean;
	hasDarkLogo: boolean;
	requiresReauthentication: boolean;
	launchURL?: string;
	clientType: OidcClientType;
};

export type OidcClientFederatedIdentity = {
	issuer: string;
	subject?: string;
	audience?: string;
	jwks?: string | undefined;
	publicKeys?: Jwk[];
	replayProtection: boolean;
};

export type OidcClientSecret = {
	id: string;
	// The first characters of the secret, empty for secrets created before Pocket ID supported multiple secrets
	prefix: string;
	createdAt: string;
	expiresAt: string | null;
	isActive: boolean;
};

// The clear-text value of a secret is only returned when it is created and cannot be retrieved afterwards
export type OidcClientSecretCreated = OidcClientSecret & {
	secret: string;
};

export type OidcClientCredentials = {
	federatedIdentities: OidcClientFederatedIdentity[];
	secrets: OidcClientSecret[];
};

export type OidcDiscoveryConfiguration = {
	issuer: string;
	authorization_endpoint: string;
	token_endpoint: string;
	userinfo_endpoint: string;
	end_session_endpoint: string;
	jwks_uri: string;
};

export type OidcClient = OidcClientMetaData & {
	callbackURLs: string[];
	logoutCallbackURLs: string[];
	backchannelLogoutURL: string;
	isPublic: boolean;
	pkceEnabled: boolean;
	requiresReauthentication: boolean;
	requiresPushedAuthorizationRequests: boolean;
	skipConsent: boolean;
	credentials?: OidcClientCredentials;
	launchURL?: string;
	isGroupRestricted: boolean;
	pkceSupported: boolean;
	accessTokenDurationMinutes: number;
	refreshTokenDurationMinutes: number;
};

export type OidcClientTokenLifetimes = Pick<
	OidcClient,
	'accessTokenDurationMinutes' | 'refreshTokenDurationMinutes'
>;

export type OidcClientWithAllowedUserGroups = OidcClient & {
	allowedUserGroups: UserGroup[];
};

export type OidcClientWithAllowedGroups = OidcClient & {
	allowedUserGroups: UserGroupMinimal[];
};

export type OidcClientUpdate = Omit<
	OidcClient,
	'id' | 'logoURL' | 'hasLogo' | 'hasDarkLogo' | 'pkceSupported' | 'clientType'
>;
export type OidcClientCreate = OidcClientUpdate & {
	id?: string;
};
export type OidcClientUpdateWithLogo = OidcClientUpdate & {
	logo: File | null | undefined;
	darkLogo: File | null | undefined;
};

export type OidcClientCreateWithLogo = OidcClientCreate & {
	logo?: File | null;
	logoUrl?: string;
	darkLogo?: File | null;
	darkLogoUrl?: string;
};

export type OidcDeviceCodeInfo = {
	scope: string[];
	scopeInfo: InteractionScopeInfo[];
	authorizationRequired: boolean;
	reauthenticationRequired: boolean;
	client: OidcClientMetaData;
};

export type AccessibleOidcClient = OidcClientMetaData & {
	lastUsedAt: Date | null;
};

export type AuthorizedOidcClient = {
	scope: string;
	client: OidcClientMetaData;
	lastUsedAt: Date;
};

export type InteractionStep = 'authenticate' | 'select_account' | 'reauthenticate' | 'consent';

export type InteractionScopeInfo = {
	key: string;
	name: string;
	description?: string;
};

export type InteractionSession = {
	id: string;
	scopes: string[];
	scopeInfo: InteractionScopeInfo[];
	client: OidcClientMetaData;
	currentStep?: InteractionStep;
	requiredSteps: InteractionStep[];
};

export type CompleteInteractionResponse = {
	interaction?: InteractionSession;
	redirectUrl?: string;
};

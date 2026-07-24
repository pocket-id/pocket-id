import type { UserGroup } from './user-group.type';

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
	clientIdHost?: string;
};

export type OidcClientFederatedIdentity = {
	issuer: string;
	subject?: string;
	audience?: string;
	jwks?: string | undefined;
	replayProtection: boolean;
};

export type OidcClientCredentials = {
	federatedIdentities: OidcClientFederatedIdentity[];
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
	isPublic: boolean;
	pkceEnabled: boolean;
	requiresReauthentication: boolean;
	requiresPushedAuthorizationRequests: boolean;
	skipConsent: boolean;
	credentials?: OidcClientCredentials;
	launchURL?: string;
	isGroupRestricted: boolean;
	pkceSupported: boolean;
};

export type OidcClientWithAllowedUserGroups = OidcClient & {
	allowedUserGroups: UserGroup[];
};

export type OidcClientWithAllowedUserGroupsCount = OidcClient & {
	allowedUserGroupsCount: number;
};

export type OidcClientUpdate = Omit<
	OidcClient,
	'id' | 'logoURL' | 'hasLogo' | 'hasDarkLogo' | 'pkceSupported' | 'clientType' | 'clientIdHost'
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

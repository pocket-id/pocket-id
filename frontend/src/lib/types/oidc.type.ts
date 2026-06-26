import type { UserGroup } from './user-group.type';

export type OidcClientMetaData = {
	id: string;
	name: string;
	hasLogo: boolean;
	hasDarkLogo: boolean;
	requiresReauthentication: boolean;
	launchURL?: string;
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

export type OidcClient = OidcClientMetaData & {
	callbackURLs: string[];
	logoutCallbackURLs: string[];
	isPublic: boolean;
	pkceEnabled: boolean;
	requiresReauthentication: boolean;
	requiresPushedAuthorizationRequests: boolean;
	credentials?: OidcClientCredentials;
	launchURL?: string;
	isGroupRestricted: boolean;
};

export type OidcClientWithAllowedUserGroups = OidcClient & {
	allowedUserGroups: UserGroup[];
};

export type OidcClientWithAllowedUserGroupsCount = OidcClient & {
	allowedUserGroupsCount: number;
};

export type OidcClientUpdate = Omit<OidcClient, 'id' | 'logoURL' | 'hasLogo' | 'hasDarkLogo'>;
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

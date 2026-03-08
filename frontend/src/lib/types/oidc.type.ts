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
	scope: string;
	authorizationRequired: boolean;
	client: OidcClientMetaData;
};

export type AuthorizeResponse = {
	code?: string;
	callbackURL: string;
	issuer?: string;
	error?: string;
	requiresRedirect?: boolean;
};

export type AccessibleOidcClient = OidcClientMetaData & {
	lastUsedAt: Date | null;
};

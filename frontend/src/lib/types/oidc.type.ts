import type { UserGroup } from './user-group.type';

export type OidcClientMetaData = {
	id: string;
	name: string;
	hasLogo: boolean;
};

export type OidcClientFederatedIdentity = {
	issuer: string;
	subject?: string;
	audience?: string;
	jwks: string | undefined;
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
};

export type OidcClientWithAllowedUserGroups = OidcClient & {
	allowedUserGroups: UserGroup[];
};

export type OidcClientWithAllowedUserGroupsCount = OidcClient & {
	allowedUserGroupsCount: number;
};

export type OidcClientCreate = Omit<OidcClient, 'id' | 'logoURL' | 'hasLogo'>;

export type OidcClientCreateWithLogo = OidcClientCreate & {
	logo: File | null | undefined;
};

export type OidcDeviceCodeInfo = {
	scope: string;
	authorizationRequired: boolean;
	client: OidcClientMetaData;
};

export type AuthorizeResponse = {
	code: string;
	callbackURL: string;
	issuer: string;
};

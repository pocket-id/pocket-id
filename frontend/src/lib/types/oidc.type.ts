import type { UserGroup } from './user-group.type';

export type OidcClientMetaData = {
	id: string;
	name: string;
	hasLogo: boolean;
};

export type OidcClient = OidcClientMetaData & {
	callbackURLs: [string, ...string[]];
	logoutCallbackURLs: string[];
	isPublic: boolean;
	pkceEnabled: boolean;
	deviceCodeEnabled: boolean;
};

export type OidcClientWithAllowedUserGroups = OidcClient & {
	allowedUserGroups: UserGroup[];
};

export type OidcClientCreate = Omit<OidcClient, 'id' | 'logoURL' | 'hasLogo'>;

export type OidcClientCreateWithLogo = OidcClientCreate & {
	logo: File | null | undefined;
};

export type AuthorizeResponse = {
	code: string;
	callbackURL: string;
};

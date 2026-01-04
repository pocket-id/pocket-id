import type { OidcClientMetaData } from './oidc.type';

export type ScimServiceProvider = {
	id: string;
	endpoint: string;
	token?: string;
	lastSyncedAt?: string;
	createdAt: string;
	oidcClient: OidcClientMetaData;
};

export type ScimServiceProviderCreate = Pick<ScimServiceProvider, 'endpoint' | 'token'> & {
	oidcClientId: string;
};

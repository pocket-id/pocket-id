export type ScimServiceProvider = {
	id: string;
	endpoint: string;
	hasToken: boolean;
	lastSyncedAt?: string;
	createdAt: string;
};

export type ScimServiceProviderCreate = {
	endpoint: string;
	token?: string;
	oidcClientId: string;
};

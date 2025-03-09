export type ApiKey = {
	id: string;
	name: string;
	description?: string;
	expiresAt: string;
	lastUsedAt?: string;
	createdAt: string;
};

export type ApiKeyCreate = {
	name: string;
	description?: string;
	expiresAt:  Date;
};

export type ApiKeyResponse = {
	apiKey: ApiKey;
	token: string;
};

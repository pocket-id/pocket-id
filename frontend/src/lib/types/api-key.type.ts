export type ApiKey = {
	id: string;
	name: string;
	description?: string;
	expiresAt: string;
	lastUsedAt?: string;
	createdAt: string;
	scopes: string[];
	previousKeyExpiresAt?: string;
};

export type ApiKeyCreate = {
	name: string;
	description?: string;
	expiresAt: Date;
	scopes: string[];
};

export type ApiKeyUpdate = Omit<ApiKeyCreate, 'expiresAt'>;

export type ApiKeyResponse = {
	apiKey: ApiKey;
	token: string;
};

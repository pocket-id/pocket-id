export type RuntimeCredential = {
	id: string;
	name: string;
	algorithm: string;
	createdAt: string;
	lastUsedAt?: string;
	expiresAt?: string;
	revokedAt?: string;
};

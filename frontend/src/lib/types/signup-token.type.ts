import type { UserGroup } from './user-group.type';

export interface SignupToken {
	id: string;
	// Only set in the response that creates the token
	token?: string;
	prefix: string;
	expiresAt: string;
	usageLimit: number;
	usageCount: number;
	userGroups: UserGroup[];
	createdAt: string;
	sentTo?: string;
}

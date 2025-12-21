import type { UserGroup } from './user-group.type';

export interface SignupToken {
	id: string;
	token: string;
	expiresAt: string;
	usageLimit: number;
	usageCount: number;
	userGroups: UserGroup[];
	createdAt: string;
}

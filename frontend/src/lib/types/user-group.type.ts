import type { CustomClaim } from './custom-claim.type';
import type { User } from './user.type';

export type UserGroup = {
	id: string;
	friendlyName: string;
	name: string;
	createdAt: string;
	customClaims: CustomClaim[];
	ldapAttributes: CustomClaim[];
	ldapId?: string;
};

export type UserGroupWithUsers = UserGroup & {
	users: User[];
};

export type UserGroupWithUserCount = UserGroup & {
	userCount: number;
};

export type UserGroupCreate = Pick<UserGroup, 'friendlyName' | 'name' | 'ldapId'>;

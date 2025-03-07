import type { CustomClaim } from './custom-claim.type';
import type { UserGroup } from './user-group.type';

export type User = {
	id: string;
	username: string;
	email: string;
	firstName: string;
	lastName: string;
	isAdmin: boolean;
	userGroups: UserGroup[];
	customClaims: CustomClaim[];
	ldapId?: string;
};

export type UserCreate = Omit<User, 'id' | 'customClaims' | 'ldapId' | 'userGroups'>;

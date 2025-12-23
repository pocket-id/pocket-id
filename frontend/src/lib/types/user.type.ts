import type { Locale } from '$lib/paraglide/runtime';
import type { CustomClaim } from './custom-claim.type';
import type { UserGroup } from './user-group.type';

export type User = {
	id: string;
	username: string;
	email: string | undefined;
	firstName: string;
	lastName?: string;
	displayName: string;
	birthDate?: Date;
	isAdmin: boolean;
	userGroups: UserGroup[];
	customClaims: CustomClaim[];
	locale?: Locale;
	ldapId?: string;
	disabled?: boolean;
};

export type UserCreate = Omit<User, 'id' | 'customClaims' | 'ldapId' | 'userGroups'>;

export type UserSignUp = Omit<UserCreate, 'isAdmin' | 'disabled' | 'displayName'> & {
	token?: string;
};

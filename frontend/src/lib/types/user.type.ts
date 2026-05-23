import type { Locale } from '$lib/paraglide/runtime';
import type { CustomFieldValue } from './custom-field.type';
import type { UserGroup } from './user-group.type';

export type User = {
	id: string;
	username: string;
	email: string | undefined;
	emailVerified: boolean;
	firstName: string;
	lastName?: string;
	displayName: string;
	isAdmin: boolean;
	userGroups: UserGroup[];
	customFieldValues: CustomFieldValue[];
	locale?: Locale;
	ldapId?: string;
	disabled?: boolean;
};

export type UserCreate = Omit<User, 'id' | 'customFieldValues' | 'ldapId' | 'userGroups'> & {
	customFieldValues?: CustomFieldValue[];
};

export type AccountUpdate = Omit<
	UserCreate,
	'isAdmin' | 'disabled' | 'emailVerified'
>;

export type UserSignUp = Omit<
	UserCreate,
	'isAdmin' | 'disabled' | 'displayName' | 'emailVerified'
> & {
	token?: string;
};

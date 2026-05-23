import type { CustomFieldValue } from './custom-field.type';
import type { OidcClientMetaData } from './oidc.type';
import type { User } from './user.type';

export type UserGroup = {
	id: string;
	friendlyName: string;
	name: string;
	createdAt: string;
	customFieldValues: CustomFieldValue[];
	ldapId?: string;
	users: User[];
	allowedOidcClients: OidcClientMetaData[];
};

export type UserGroupMinimal = Omit<UserGroup, 'users' | 'allowedOidcClients'> & {
	userCount: number;
};

export type UserGroupCreate = Pick<UserGroup, 'friendlyName' | 'name' | 'ldapId'> & {
	customFieldValues?: CustomFieldValue[];
};

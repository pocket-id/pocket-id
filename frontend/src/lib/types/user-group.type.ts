import type { CustomClaim } from './custom-claim.type';
import type { OidcClientMetaData } from './oidc.type';
import type { User } from './user.type';

export type UserGroup = {
	id: string;
	friendlyName: string;
	name: string;
	createdAt: string;
	customClaims: CustomClaim[];
	ldapId?: string;
	users: User[];
	allowedOidcClients: OidcClientMetaData[];
};

export type UserGroupMinimal = Omit<UserGroup, 'users' | 'allowedOidcClients'> & {
	userCount: number;
};

export type UserGroupCreate = Pick<UserGroup, 'friendlyName' | 'name' | 'ldapId'>;

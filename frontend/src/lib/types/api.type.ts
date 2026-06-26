export type ApiPermission = {
	id: string;
	key: string;
	name: string;
	description?: string;
};

export type Api = {
	id: string;
	name: string;
	audience: string;
	createdAt: string;
	permissions: ApiPermission[];
};

export type ApiListItem = Omit<Api, 'permissions'> & {
	permissionCount: number;
};

export type ApiCreate = {
	name: string;
	audience: string;
};

export type ApiUpdate = {
	name: string;
};

export type ApiPermissionInput = {
	key: string;
	name: string;
	description: string;
};

export type ClientApiAccess = {
	allowedPermissionIds: string[];
};

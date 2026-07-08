export type ApiPermission = {
	id: string;
	key: string;
	name: string;
	description?: string;
};

export type Api = {
	id: string;
	name: string;
	resource: string;
	createdAt: string;
	permissions: ApiPermission[];
};

export type ApiCreate = {
	name: string;
	resource: string;
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
	userDelegatedPermissionIds: string[];
	clientPermissionIds: string[];
};

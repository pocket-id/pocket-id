export type ApiPermission = {
	id: string;
	key: string;
	name: string;
	description?: string;
	allowedForCimdClients: boolean;
};

export type Api = {
	id: string;
	name: string;
	resource: string;
	createdAt: string;
	permissions: ApiPermission[];
	allowCimdClients: boolean;
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

export type ApiClientGrant = {
	userDelegatedAccess: boolean;
	clientAccess: boolean;
	userDelegatedPermissionIds: string[];
	clientPermissionIds: string[];
};

export type ApiCimdAccessUpdate = {
	enabled: boolean;
	permissionIds: string[];
};

export type ApiClient = {
	id: string;
	name: string;
	clientType: string;
	isPublic: boolean;
	hasLogo: boolean;
	hasDarkLogo: boolean;
};

export type ApiClientAccess = ApiClientGrant & {
	client: ApiClient;
	cimdGrantedAccess: boolean;
	cimdGrantedPermissionIds: string[];
};

export type ClientApiGrant = ApiClientGrant & {
	api: Api;
	cimdGrantedAccess: boolean;
	cimdGrantedPermissionIds: string[];
};

import type {
	Api,
	ApiCimdAccessUpdate,
	ApiClient,
	ApiClientAccess,
	ApiClientGrant,
	ApiCreate,
	ApiPermissionInput,
	ApiUpdate,
	ClientApiGrant
} from '$lib/types/api.type';
import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import { encodeClientIdParam } from '$lib/utils/client-id-util';
import APIService from './api-service';

export default class ApisService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/apis', { params: options });
		return res.data as Paginated<Api>;
	};

	listAll = async () => {
		const res = await this.api.get('/apis', { params: { pagination: { page: 1, limit: 1000 } } });
		return (res.data as Paginated<Api>).data;
	};

	get = async (id: string) => {
		const res = await this.api.get(`/apis/${id}`);
		return res.data as Api;
	};

	create = async (api: ApiCreate) => {
		const res = await this.api.post('/apis', api);
		return res.data as Api;
	};

	update = async (id: string, api: ApiUpdate) => {
		const res = await this.api.put(`/apis/${id}`, api);
		return res.data as Api;
	};

	remove = async (id: string) => {
		await this.api.delete(`/apis/${id}`);
	};

	updatePermissions = async (id: string, permissions: ApiPermissionInput[]) => {
		const res = await this.api.put(`/apis/${id}/permissions`, { permissions });
		return res.data as Api;
	};

	updateCimdAccess = async (id: string, access: ApiCimdAccessUpdate) => {
		const res = await this.api.put(`/apis/${id}/cimd-access`, access);
		return res.data as Api;
	};

	listClients = async (id: string, options?: ListRequestOptions) => {
		const res = await this.api.get(`/apis/${id}/clients`, { params: options });
		return res.data as Paginated<ApiClientAccess>;
	};

	listAssignableClients = async (id: string, options?: ListRequestOptions) => {
		const res = await this.api.get(`/apis/${id}/assignable-clients`, { params: options });
		return res.data as Paginated<ApiClient>;
	};

	updateClientAccessForApi = async (id: string, clientId: string, grant: ApiClientGrant) => {
		const res = await this.api.put(`/apis/${id}/clients/${encodeClientIdParam(clientId)}`, grant);
		return res.data as ApiClientGrant;
	};

	removeClientAccessForApi = async (id: string, clientId: string) => {
		await this.api.delete(`/apis/${id}/clients/${encodeClientIdParam(clientId)}`);
	};

	listClientApis = async (clientId: string) => {
		const res = await this.api.get(`/api-access/${encodeClientIdParam(clientId)}/apis`);
		return res.data as ClientApiGrant[];
	};

	listAssignableApis = async (clientId: string, options?: ListRequestOptions) => {
		const res = await this.api.get(`/api-access/${encodeClientIdParam(clientId)}/assignable-apis`, {
			params: options
		});
		return res.data as Paginated<Api>;
	};
}

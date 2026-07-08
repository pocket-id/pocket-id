import type {
	Api,
	ApiCreate,
	ApiPermissionInput,
	ApiUpdate,
	ClientApiAccess
} from '$lib/types/api.type';
import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
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

	getClientAccess = async (clientId: string) => {
		const res = await this.api.get(`/api-access/${clientId}`);
		return res.data as ClientApiAccess;
	};

	updateClientAccess = async (clientId: string, access: ClientApiAccess) => {
		const res = await this.api.put(`/api-access/${clientId}`, access);
		return res.data as ClientApiAccess;
	};
}

import type { ApiKey, ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import APIService from './api-service';

export default class ApiKeyService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/api-keys', { params: options });
		return res.data as Paginated<ApiKey>;
	};

	create = async (data: ApiKeyCreate): Promise<ApiKeyResponse> => {
		const res = await this.api.post('/api-keys', data);
		return res.data as ApiKeyResponse;
	};

	renew = async (id: string, expiresAt: Date): Promise<ApiKeyResponse> => {
		const res = await this.api.post(`/api-keys/${id}/renew`, {
			expiresAt
		});
		return res.data as ApiKeyResponse;
	};

	revoke = async (id: string): Promise<void> => {
		await this.api.delete(`/api-keys/${id}`);
	};
}

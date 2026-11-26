import type {
	ApiKey,
	ApiKeyCreate,
	ApiKeyResponse,
	ApiKeyUpdate
} from '#lib/types/api-key.type.ts';
import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
import APIService from './api-service';
import { pathSegment } from '#lib/utils/path-segment-util.ts';

export default class ApiKeyService extends APIService {
	list = async (options?: ListRequestOptions, signal?: AbortSignal) => {
		const res = await this.api.get('/api-keys', { params: options, signal });
		return res.data as Paginated<ApiKey>;
	};

	create = async (data: ApiKeyCreate): Promise<ApiKeyResponse> => {
		const res = await this.api.post('/api-keys', data);
		return res.data as ApiKeyResponse;
	};

	update = async (id: string, data: ApiKeyUpdate): Promise<ApiKey> => {
		const res = await this.api.put(`/api-keys/${pathSegment(id)}`, data);
		return res.data as ApiKey;
	};

	renew = async (
		id: string,
		expiresAt: Date,
		gracePeriodHours: number
	): Promise<ApiKeyResponse> => {
		const res = await this.api.post(`/api-keys/${pathSegment(id)}/renew`, {
			expiresAt,
			gracePeriodHours
		});
		return res.data as ApiKeyResponse;
	};

	revoke = async (id: string): Promise<void> => {
		await this.api.delete(`/api-keys/${pathSegment(id)}`);
	};
}

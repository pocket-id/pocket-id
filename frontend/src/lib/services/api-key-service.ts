import type { ApiKey, ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
import type { Paginated } from '$lib/types/pagination.type';
import type { SearchPaginationSortRequest } from '$lib/types/sort-pagination.type';
import APIService from './api-service';

export default class ApiKeyService extends APIService {
	async list(options?: SearchPaginationSortRequest): Promise<Paginated<ApiKey>> {
		const queryParams = new URLSearchParams();

		if (options?.search) {
			queryParams.append('search', options.search);
		}

		if (options?.pagination) {
			queryParams.append('page', options.pagination.page.toString());
			queryParams.append('limit', options.pagination.limit.toString());
		}

		if (options?.sort) {
			queryParams.append('sort', options.sort.column);
			queryParams.append('direction', options.sort.direction);
		}

		const query = queryParams.toString() ? `?${queryParams.toString()}` : '';
		const res = await this.api.get(`/api-keys${query}`);
		return res.data;
	}

	async create(apiKey: ApiKeyCreate): Promise<ApiKeyResponse> {
		const payload = {
			...apiKey,
			expiresAt:
				apiKey.expiresAt instanceof Date ? apiKey.expiresAt.toISOString() : apiKey.expiresAt
		};

		const res = await this.api.post('/api-keys', payload);
		return res.data as ApiKeyResponse;
	}

	async revoke(id: string): Promise<void> {
		await this.api.delete(`/api-keys/${id}`);
	}
}

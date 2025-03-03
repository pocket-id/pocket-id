import type { ApiKey, ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
import APIService from './api-service';

export default class ApiKeyService extends APIService {
	async list() {
		const res = await this.api.get('/api-keys');
		return res.data as ApiKey[];
	}

	async create(apiKey: ApiKeyCreate) {
		const res = await this.api.post('/api-keys', apiKey);
		return res.data as ApiKeyResponse;
	}

	async revoke(id: string) {
		await this.api.delete(`/api-keys/${id}`);
	}
}

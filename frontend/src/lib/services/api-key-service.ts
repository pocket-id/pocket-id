import type { ApiKey, ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import APIService from './api-service';

export default class ApiKeyService extends APIService {
  list = async (options?: SearchPaginationSortRequest) => {
    const res = await this.api.get('/api-keys', { params: options });
    return res.data as Paginated<ApiKey>;
  };

  create = async (data: ApiKeyCreate): Promise<ApiKeyResponse> => {
    const res = await this.api.post('/api-keys', data);
    return res.data as ApiKeyResponse;
  };

  revoke = async (id: string): Promise<void> => {
    await this.api.delete(`/api-keys/${id}`);
  };
}

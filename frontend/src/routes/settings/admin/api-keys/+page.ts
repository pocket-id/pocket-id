import ApiKeyService from '$lib/services/api-key-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const apiKeyService = new ApiKeyService();

	const apiKeysRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'lastUsedAt',
			direction: 'desc' as const
		}
	};

	const apiKeys = await apiKeyService.list(apiKeysRequestOptions);

	return { apiKeys, apiKeysRequestOptions };
};

import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import ApiKeyService from '$lib/services/api-key-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const apiKeyService = new ApiKeyService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const apiKeysRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'lastUsedAt',
			direction: 'desc' as const
		}
	};

	const apiKeys = await apiKeyService.list(apiKeysRequestOptions);

	return { apiKeys, apiKeysRequestOptions };
};

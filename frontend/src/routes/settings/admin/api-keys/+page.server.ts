import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import ApiKeyService from '$lib/services/api-key-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const apiKeyService = new ApiKeyService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const apiKeys = await apiKeyService.list({
		sort: {
			column: 'lastUsedAt',
			direction: 'desc' as const
		}
	});

	return apiKeys;
};

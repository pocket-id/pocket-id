import type { PageServerLoad } from './$types';
import ApiKeyService from '$lib/services/api-key-service';
import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';

export const load: PageServerLoad = async ({ cookies }) => {
	// Create service with auth token from cookies
	const apiKeyService = new ApiKeyService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	// Request options with default sorting
	const requestOptions = {
		sort: {
			column: 'name',
			direction: 'asc' as const
		},
		pagination: {
			page: 1,
			limit: 10
		}
	};

	// Fetch API keys (the service will use the token for auth)
	const apiKeys = await apiKeyService.list(requestOptions);

	return apiKeys;
};

import ApiKeyService from '#lib/services/api-key-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const apiKeys = await new ApiKeyService().list({ pagination: { page: 1, limit: 1 } });
	return { empty: apiKeys.pagination.totalItems === 0 };
};

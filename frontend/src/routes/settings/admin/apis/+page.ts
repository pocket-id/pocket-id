import ApisService from '#lib/services/apis-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const apis = await new ApisService().list({ pagination: { page: 1, limit: 1 } });
	return { empty: apis.pagination.totalItems === 0 };
};

import ApisService from '#lib/services/apis-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const apisService = new ApisService();
	const [api, clients] = await Promise.all([
		apisService.get(params.id),
		apisService.listClients(params.id, { pagination: { page: 1, limit: 100 } })
	]);
	return { api, clients: clients.data };
};

import OIDCService from '#lib/services/oidc-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const clients = await new OIDCService().listClients({ pagination: { page: 1, limit: 1 } });
	return { empty: clients.pagination.totalItems === 0 };
};

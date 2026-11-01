import OidcService from '#lib/services/oidc-service.ts';
import UserGroupService from '#lib/services/user-group-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const [userGroup, clients] = await Promise.all([
		new UserGroupService().get(params.id),
		// Every client, to name the ones open to everyone and to warn before a client loses its last group
		new OidcService().listClients({
			pagination: { page: 1, limit: 100 },
			sort: { column: 'name', direction: 'asc' }
		})
	]);
	return { userGroup, clients: clients.data };
};

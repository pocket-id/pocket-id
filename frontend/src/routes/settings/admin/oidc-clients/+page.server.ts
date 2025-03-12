import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const oidcService = new OIDCService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const clientsRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'name',
			direction: 'asc'
		}
	};

	const clients = await oidcService.listClients(clientsRequestOptions);

	return { clients, clientsRequestOptions };
};

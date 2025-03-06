import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const oidcService = new OIDCService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	// Create request options with default sorting
	const requestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'name',
			direction: 'asc'
		},
		pagination: {
			page: 1,
			limit: 10
		}
	};

	const clients = await oidcService.listClients(requestOptions);

	return clients;
};

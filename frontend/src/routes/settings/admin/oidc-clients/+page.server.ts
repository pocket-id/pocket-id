import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import OIDCService from '$lib/services/oidc-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, url }) => {
	const oidcService = new OIDCService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	// Get sort parameters from URL or use defaults
	const sortColumn = url.searchParams.get('sort') || 'name';
	const sortDirection = url.searchParams.get('direction') || 'asc';

	// Create request options with default sorting
	const requestOptions = {
		sort: {
			column: sortColumn,
			direction: sortDirection as 'asc' | 'desc'
		},
		pagination: {
			page: 1,
			limit: 10
		}
	};

	const clients = await oidcService.listClients(requestOptions);

	return clients;
};

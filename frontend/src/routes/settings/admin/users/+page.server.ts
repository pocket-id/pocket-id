import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserService from '$lib/services/user-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, url }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	// Get sort parameters from URL or use defaults
	const sortColumn = 'firstName';
	const sortDirection = 'asc';

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

	const users = await userService.list(requestOptions);
	return users;
};

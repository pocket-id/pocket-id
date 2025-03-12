import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserService from '$lib/services/user-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const usersRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'firstName',
			direction: 'asc'
		}
	};

	const users = await userService.list(usersRequestOptions);
	return {users, usersRequestOptions};
};

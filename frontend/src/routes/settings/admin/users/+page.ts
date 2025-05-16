import UserService from '$lib/services/user-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();

	const usersRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'firstName',
			direction: 'asc'
		}
	};

	const users = await userService.list(usersRequestOptions);
	return { users, usersRequestOptions };
};

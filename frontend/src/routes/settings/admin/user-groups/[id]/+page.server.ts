import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserGroupService from '$lib/services/user-group-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies }) => {
	const userGroupService = new UserGroupService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

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

	const userGroup = await userGroupService.get(params.id, requestOptions);

	return { userGroup };
};

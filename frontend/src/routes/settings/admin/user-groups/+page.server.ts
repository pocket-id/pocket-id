import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserGroupService from '$lib/services/user-group-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const userGroupService = new UserGroupService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const userGroupsRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'friendlyName',
			direction: 'asc'
		},
	};

	const userGroups = await userGroupService.list(userGroupsRequestOptions);
	return {userGroups, userGroupsRequestOptions};
};

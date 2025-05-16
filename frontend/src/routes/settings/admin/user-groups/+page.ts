import UserGroupService from '$lib/services/user-group-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userGroupService = new UserGroupService();

	const userGroupsRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'friendlyName',
			direction: 'asc'
		}
	};

	const userGroups = await userGroupService.list(userGroupsRequestOptions);
	return { userGroups, userGroupsRequestOptions };
};

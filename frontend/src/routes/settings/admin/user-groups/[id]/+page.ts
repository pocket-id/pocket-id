import UserGroupService from '$lib/services/user-group-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const userGroupService = new UserGroupService();
	const userGroup = await userGroupService.get(params.id);

	return { userGroup };
};

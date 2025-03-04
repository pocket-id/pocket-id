import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import UserService from '$lib/services/user-service';
import UserGroupService from '$lib/services/user-group-service';

export const load: PageServerLoad = async ({ params, cookies, fetch }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
	const userGroupService = new UserGroupService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	try {
		// Get the user
		const user = await userService.get(params.id);

		// Get all user groups
		const userGroups = await userGroupService.list();

		// Get the user's group memberships
		const userGroupMemberships = await Promise.all(
			userGroups.data.map(async (group) => {
				const groupDetails = await userGroupService.get(group.id);
				return {
					...group,
					hasMember: groupDetails.users.some((u) => u.id === params.id)
				};
			})
		);

		// Add user's groups as a property
		user.userGroups = userGroups.data.filter((group) =>
			userGroupMemberships.find((membership) => membership.id === group.id && membership.hasMember)
		);

		// Add all groups to the page props
		return {
			user,
			allUserGroups: {
				data: userGroupMemberships,
				pagination: userGroups.pagination
			}
		};
	} catch (e) {
		console.error('Error loading user data:', e);
		throw error(404, 'User not found');
	}
};

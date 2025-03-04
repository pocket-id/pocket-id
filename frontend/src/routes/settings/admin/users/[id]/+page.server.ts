import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import UserService from '$lib/services/user-service';
import UserGroupService from '$lib/services/user-group-service';

export const load: PageServerLoad = async ({ params, cookies, fetch }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
	const userGroupService = new UserGroupService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	try {
		// Get the user data
		const user = await userService.get(params.id);

		// Get user's group memberships separately since they're not included in the user object
		const userGroups = await userService.getUserGroups(params.id);

		// Get all groups with pagination
		const allGroups = await userGroupService.list();

		// Create a set of group IDs the user belongs to for quick lookups
		const userGroupIds = new Set(userGroups.map((g) => g.id));

		// Add membership info to each group without additional API calls
		const groupsWithMembership = {
			data: allGroups.data.map((group) => ({
				...group,
				hasMember: userGroupIds.has(group.id)
			})),
			pagination: allGroups.pagination
		};

		return {
			user,
			allUserGroups: groupsWithMembership,
			userGroupIds: Array.from(userGroupIds),
			// Also include original groups for reference
			userGroups
		};
	} catch (e) {
		console.error('Error loading user data:', e);
		// Log more details about the error
		if (e instanceof Error) {
			console.error('Error message:', e.message);
			console.error('Error stack:', e.stack);
		}
		throw error(404, 'User not found');
	}
};

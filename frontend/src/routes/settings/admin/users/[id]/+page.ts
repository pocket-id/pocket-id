import UserService from '$lib/services/user-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const userService = new UserService();
	const [user, passkeys] = await Promise.all([
		userService.get(params.id),
		userService.listUserPasskeys(params.id)
	]);

	return {
		user,
		passkeys
	};
};

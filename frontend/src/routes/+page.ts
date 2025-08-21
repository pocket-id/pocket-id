import { redirect } from '@sveltejs/kit';
import UserService from '$lib/services/user-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();
	
	try {
		const user = await userService.getCurrent();
		if (user) {
			return redirect(302, '/settings/apps');
		}
	} catch (error) {
		// User not logged in, continue to login
	}
	
	return redirect(302, '/login');
};

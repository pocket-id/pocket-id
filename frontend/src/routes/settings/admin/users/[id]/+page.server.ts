import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserService from '$lib/services/user-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params, cookies }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const user = await userService.get(params.id);

	return {
		user
	};
};

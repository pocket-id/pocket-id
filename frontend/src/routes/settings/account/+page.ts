import UserService from '#lib/services/user-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();
	const account = await userService.getCurrent();

	return {
		account
	};
};

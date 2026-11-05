import SessionService from '#lib/services/session-service.ts';
import UserService from '#lib/services/user-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const userService = new UserService();
	const [user, passkeys, sessions] = await Promise.all([
		userService.get(params.id),
		userService.listUserPasskeys(params.id),
		new SessionService().listForUser(params.id)
	]);

	return {
		user,
		passkeys,
		sessions
	};
};

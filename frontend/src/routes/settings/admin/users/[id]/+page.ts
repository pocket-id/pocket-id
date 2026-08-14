import UserService from '$lib/services/user-service';
import RuntimeCredentialService from '$lib/services/runtime-credential-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const userService = new UserService();
	const runtimeCredentialService = new RuntimeCredentialService();
	const [user, passkeys, runtimeCredentials] = await Promise.all([
		userService.get(params.id),
		userService.listUserPasskeys(params.id),
		runtimeCredentialService.listForUser(params.id)
	]);

	return {
		user,
		passkeys,
		runtimeCredentials
	};
};

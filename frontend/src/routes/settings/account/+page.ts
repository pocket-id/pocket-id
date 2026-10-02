import UserService from '#lib/services/user-service.ts';
import WebAuthnService from '#lib/services/webauthn-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const webauthnService = new WebAuthnService();
	const userService = new UserService();

	const [account, passkeys] = await Promise.all([
		userService.getCurrent(),
		webauthnService.listCredentials()
	]);

	return {
		account,
		passkeys
	};
};

import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import UserService from '$lib/services/user-service';
import WebAuthnService from '$lib/services/webauthn-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	const webauthnService = new WebAuthnService(accessToken);
	const userService = new UserService(accessToken);

	const [account, passkeys] = await Promise.all([
		userService.getCurrent(),
		webauthnService.listCredentials()
	]);

	return {
		account,
		passkeys
	};
};

import UserService from '$lib/services/user-service';
import WebAuthnService from '$lib/services/webauthn-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const webauthnService = new WebAuthnService();
	const userService = new UserService();

	const [account, passkeys, recoveryCodeStatus] = await Promise.all([
		userService.getCurrent(),
		webauthnService.listCredentials(),
		userService.getRecoveryCodeStatus().catch(() => ({ total: 0, unused: 0 }))
	]);

	return {
		account,
		passkeys,
		recoveryCodeStatus
	};
};

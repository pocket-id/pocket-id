import UserService from '$lib/services/user-service';
import WebAuthnService from '$lib/services/webauthn-service';
import RuntimeCredentialService from '$lib/services/runtime-credential-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const webauthnService = new WebAuthnService();
	const userService = new UserService();
	const runtimeCredentialService = new RuntimeCredentialService();

	const account = await userService.getCurrent();
	const passkeys = account.isAgent ? [] : await webauthnService.listCredentials();
	const runtimeCredentials = account.isAgent ? await runtimeCredentialService.list() : [];

	return {
		account,
		passkeys,
		runtimeCredentials
	};
};

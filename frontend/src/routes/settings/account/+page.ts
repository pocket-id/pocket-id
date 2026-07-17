import UserService from '$lib/services/user-service';
import WebAuthnService from '$lib/services/webauthn-service';
import OIDCService from '$lib/services/oidc-service';
import type { ListRequestOptions } from '$lib/types/list-request.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const webauthnService = new WebAuthnService();
	const userService = new UserService();
	const oidcService = new OIDCService();

	const clientRequestOptions: ListRequestOptions = { sort: { column: 'lastUsedAt', direction: 'desc' } };
	const [account, passkeys, accessibleClients] = await Promise.all([
		userService.getCurrent(),
		webauthnService.listCredentials(),
		oidcService.listOwnAccessibleClients(clientRequestOptions).catch(() => null)
	]);

	return {
		account,
		passkeys,
		accessibleClients
	};
};

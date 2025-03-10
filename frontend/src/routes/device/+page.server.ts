import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import OidcService from '$lib/services/oidc-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, cookies }) => {
	// Get code and client_id from query params
	const code = url.searchParams.get('code');
	const clientId = url.searchParams.get('client_id');

	if (clientId) {
		const oidcService = new OidcService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
		const client = await oidcService.getClient(clientId);
		return {
			code,
			client,
			mode: 'authorize'
		};
	}

	return {
		code,
		mode: 'verify'
	};
};

import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import OidcService from '$lib/services/oidc-service';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, cookies }) => {
	const clientId = url.searchParams.get('client_id');
	const oidcService = new OidcService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const client = await oidcService.getClientMetaData(clientId!);

	return {
		scope: url.searchParams.get('scope')!,
		nonce: url.searchParams.get('nonce') || undefined,
		state: url.searchParams.get('state')!,
		callbackURL: url.searchParams.get('redirect_uri')!,
		client,
		codeChallenge: url.searchParams.get('code_challenge')!,
		codeChallengeMethod: url.searchParams.get('code_challenge_method')!
	};
};

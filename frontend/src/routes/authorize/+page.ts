import OidcService from '$lib/services/oidc-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const clientId = url.searchParams.get('client_id');
	const oidcService = new OidcService();

	const client = await oidcService.getClientMetaData(clientId!);

	return {
		scope: url.searchParams.get('scope')!,
		nonce: url.searchParams.get('nonce') || undefined,
		authorizeState: url.searchParams.get('state')!,
		callbackURL: url.searchParams.get('redirect_uri')!,
		client,
		codeChallenge: url.searchParams.get('code_challenge')!,
		codeChallengeMethod: url.searchParams.get('code_challenge_method')!,
		prompt: url.searchParams.get('prompt') || undefined
	};
};

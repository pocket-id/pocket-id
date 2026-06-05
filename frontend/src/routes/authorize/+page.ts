import OidcService from '$lib/services/oidc-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const clientId = url.searchParams.get('client_id');
	const requestURI = url.searchParams.get('request_uri') || undefined;
	const oidcService = new OidcService();

	const [client, parInfo] = await Promise.all([
		oidcService.getClientMetaData(clientId!),
		requestURI ? oidcService.getParRequestInfo(clientId!, requestURI) : undefined
	]);

	return {
		scope: parInfo?.scope ?? url.searchParams.get('scope')!,
		nonce: parInfo?.nonce ?? url.searchParams.get('nonce') ?? undefined,
		authorizeState: parInfo?.state ?? url.searchParams.get('state')!,
		callbackURL: parInfo?.redirectURI ?? url.searchParams.get('redirect_uri')!,
		client,
		codeChallenge: url.searchParams.get('code_challenge')!,
		codeChallengeMethod: url.searchParams.get('code_challenge_method')!,
		prompt: parInfo?.prompt ?? url.searchParams.get('prompt') ?? undefined,
		responseMode: parInfo?.responseMode ?? url.searchParams.get('response_mode') ?? undefined,
		requestURI
	};
};

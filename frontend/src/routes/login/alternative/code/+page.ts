import type { PageLoad } from './$types';
import OidcService from '$lib/services/oidc-service';

export const load: PageLoad = async ({ url }) => {

	const redirect = url.searchParams.get("redirect");

	const interactionSessionId = redirect
		? new URL(redirect, url).searchParams.get("interaction")
		: null;

	const oidcService = new OidcService();
	const interactionSession = interactionSessionId
		? await oidcService.getAuthorizeInteraction(interactionSessionId)
		: null;

	return {
		code: url.searchParams.get('code'),
		redirect: url.searchParams.get('redirect') || '/settings',
		interactionSession
	};
};
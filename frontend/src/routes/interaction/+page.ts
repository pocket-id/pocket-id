import { m } from '#lib/paraglide/messages.js';
import NetBirdService from '#lib/services/netbird-service.ts';
import OidcService from '#lib/services/oidc-service.ts';
import { getAxiosErrorMessage } from '#lib/utils/error-util.ts';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, parent }) => {
	const interactionSessionId = url.searchParams.get('interaction');
	if (!interactionSessionId) {
		return redirectToError(m.auth_missing_interaction());
	}

	const oidcService = new OidcService();
	const { appConfig } = await parent();
	const [interactionSession, netbirdIdentity] = await Promise.all([
		oidcService
			.getAuthorizeInteraction(interactionSessionId)
			.catch((e) => redirectToError(getAxiosErrorMessage(e))),
		// Only requests routed through the NetBird proxy from a known peer have an identity, so anything else signs in with a passkey
		appConfig?.netbirdEnabled ? new NetBirdService().getIdentity().catch(() => null) : null
	]);
	return {
		interactionSession,
		netbirdIdentity
	};
};

function redirectToError(errorMessage: string): never {
	redirect(302, `/interaction/error?${new URLSearchParams({ error: errorMessage }).toString()}`);
}

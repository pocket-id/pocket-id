import OidcService from '$lib/services/oidc-service';
import { getAxiosErrorMessage } from '$lib/utils/error-util';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const interactionSessionId = url.searchParams.get('interaction');
	if (!interactionSessionId) {
		return redirectToError('Missing authorize interaction');
	}

	const oidcService = new OidcService();
	const interactionSession = await oidcService
		.getAuthorizeInteraction(interactionSessionId)
		.catch((e) => redirectToError(getAxiosErrorMessage(e)));
	return {
		interactionSession
	};
};

function redirectToError(errorMessage: string): never {
	redirect(302, `/interaction/error?${new URLSearchParams({ error: errorMessage }).toString()}`);
}

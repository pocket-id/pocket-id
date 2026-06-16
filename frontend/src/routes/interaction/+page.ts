import OidcService from '$lib/services/oidc-service';
import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const interactionSessionId = url.searchParams.get('interaction');
	if (!interactionSessionId) {
		error(400, 'Missing authorize interaction');
	}

	const oidcService = new OidcService();
	const interactionSession = await oidcService.getAuthorizeInteraction(interactionSessionId);
	return {
		interactionSession
	};
};

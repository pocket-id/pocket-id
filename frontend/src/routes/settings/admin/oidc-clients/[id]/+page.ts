import OidcService from '$lib/services/oidc-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const oidcService = new OidcService();
	return await oidcService.getClient(params.id);
};

import OidcService from '$lib/services/oidc-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const oidcService = new OidcService();

	const client = await oidcService.getClient(params.id);
	const scimServiceProvider = await oidcService
		.getScimResourceProvider(params.id)
		.then((p) => p)
		.catch(() => undefined);
	return {
		client,
		scimServiceProvider
	};
};

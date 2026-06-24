import OidcService from '$lib/services/oidc-service';
import { decodeClientIdParam } from '$lib/utils/client-id-util';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const oidcService = new OidcService();
	const id = decodeClientIdParam(params.id);

	const client = await oidcService.getClient(id);
	const scimServiceProvider = await oidcService
		.getScimResourceProvider(id)
		.then((p) => p)
		.catch(() => undefined);
	return {
		client,
		scimServiceProvider
	};
};

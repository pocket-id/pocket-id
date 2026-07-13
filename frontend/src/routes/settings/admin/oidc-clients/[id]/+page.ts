import OidcService from '$lib/services/oidc-service';
import type { OidcDiscoveryConfiguration } from '$lib/types/oidc.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
	const oidcService = new OidcService();

	const clientPromise = oidcService.getClient(params.id);
	const scimServiceProviderPromise = oidcService
		.getScimResourceProvider(params.id)
		.then((p) => p)
		.catch(() => undefined);
	const oidcConfigurationPromise = fetch('/.well-known/openid-configuration').then(
		(response) => response.json() as Promise<OidcDiscoveryConfiguration>
	);

	const [client, scimServiceProvider, oidcConfiguration] = await Promise.all([
		clientPromise,
		scimServiceProviderPromise,
		oidcConfigurationPromise
	]);

	return {
		client,
		scimServiceProvider,
		oidcConfiguration
	};
};

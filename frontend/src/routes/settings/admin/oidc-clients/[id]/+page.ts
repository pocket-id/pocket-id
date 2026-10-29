import ApisService from '#lib/services/apis-service.ts';
import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
import OidcService from '#lib/services/oidc-service.ts';
import type { OidcDiscoveryConfiguration } from '#lib/types/oidc.type.ts';
import { decodeClientIdParam } from '#lib/utils/client-id-util.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
	const oidcService = new OidcService();
	const id = decodeClientIdParam(params.id);

	// The API grants load with the page, because they decide whether the Access tab's Advanced section starts open
	const [client, scimServiceProvider, oidcConfiguration, apiGrants, claimMappingPolicies] =
		await Promise.all([
			oidcService.getClient(id),
			oidcService.getScimResourceProvider(id).catch(() => undefined),
			fetch('/.well-known/openid-configuration').then(
				(response) => response.json() as Promise<OidcDiscoveryConfiguration>
			),
			new ApisService().listClientApis(id),
			new OidcClaimMappingPolicyService().listClaimMappingPolicies({
				pagination: { page: 1, limit: 100 },
				sort: { column: 'name', direction: 'asc' }
			})
		]);

	return {
		client,
		scimServiceProvider,
		oidcConfiguration,
		apiGrants,
		claimMappingPolicies: claimMappingPolicies.data
	};
};

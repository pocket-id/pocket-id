import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const policyService = new OidcClaimMappingPolicyService();
	const [policy, clients] = await Promise.all([
		policyService.getClaimMappingPolicy(params.id),
		policyService.listClients(params.id, {
			pagination: { page: 1, limit: 100 },
			sort: { column: 'name', direction: 'asc' }
		})
	]);
	return { policy, clients: clients.data };
};

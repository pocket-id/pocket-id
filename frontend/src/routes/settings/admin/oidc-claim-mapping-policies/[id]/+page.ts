import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const policy = await new OidcClaimMappingPolicyService().getClaimMappingPolicy(params.id);
	return { policy };
};

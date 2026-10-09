import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';
import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
import type {
	OidcClaimMappingPolicy,
	OidcClaimMappingPolicyDto,
	OidcClaimMappingPolicyMetadataDto
} from '#lib/types/oidc-claim-mapping-policy.type.ts';
import APIService from './api-service';

class OidcClaimMappingPolicyService extends APIService {
	listClaimMappingPolicies = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/oidc/claim-mapping-policies', {
			params: options
		});
		return res.data as Paginated<OidcClaimMappingPolicyMetadataDto>;
	};

	getClaimMappingPolicy = async (id: string) => {
		const res = await this.api.get(`/oidc/claim-mapping-policies/${id}`);
		return res.data as OidcClaimMappingPolicy;
	};

	createClaimMappingPolicy = async (policy: OidcClaimMappingPolicyDto) => {
		const res = await this.api.post('/oidc/claim-mapping-policies', policy);
		return res.data as OidcClaimMappingPolicy;
	};

	updateClaimMappingPolicy = async (id: string, policy: OidcClaimMappingPolicyDto) => {
		const res = await this.api.put(`/oidc/claim-mapping-policies/${id}`, policy);
		return res.data as OidcClaimMappingPolicy;
	};

	deleteClaimMappingPolicy = async (id: string) => {
		await this.api.delete(`/oidc/claim-mapping-policies/${id}`);
	};

	listClients = async (id: string, options?: ListRequestOptions) => {
		const res = await this.api.get(`/oidc/claim-mapping-policies/${id}/clients`, {
			params: options
		});
		return res.data as Paginated<OidcClientMetaData>;
	};

	listAssignableClients = async (id: string, options?: ListRequestOptions) => {
		const res = await this.api.get(`/oidc/claim-mapping-policies/${id}/assignable-clients`, {
			params: options
		});
		return res.data as Paginated<OidcClientMetaData>;
	};

	assignClient = async (id: string, clientId: string) => {
		await this.api.post(
			`/oidc/claim-mapping-policies/${id}/clients/${encodeClientIdParam(clientId)}`
		);
	};

	removeClient = async (id: string, clientId: string) => {
		await this.api.delete(
			`/oidc/claim-mapping-policies/${id}/clients/${encodeClientIdParam(clientId)}`
		);
	};

	duplicateClaimMappingPolicy = async (id: string, name: string) => {
		const policy = await this.getClaimMappingPolicy(id);
		const duplicatedPolicy: OidcClaimMappingPolicyDto = {
			...policy,
			id: undefined,
			name: name,
			isDefault: false
		};
		return await this.createClaimMappingPolicy(duplicatedPolicy);
	};
}

export default OidcClaimMappingPolicyService;

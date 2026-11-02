import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';
import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
import type {
	CustomScope,
	CustomScopeInput,
	OidcClaimMappingDto,
	OidcClaimMappingPolicy,
	OidcClaimMappingPolicyInput,
	OidcClaimMappingPolicyMetadataDto,
	OidcClaimMappingsPreview
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

	createClaimMappingPolicy = async (policy: OidcClaimMappingPolicyInput) => {
		const res = await this.api.post('/oidc/claim-mapping-policies', policy);
		return res.data as OidcClaimMappingPolicyInput & { id: string };
	};

	updateClaimMappingPolicy = async (id: string, policy: OidcClaimMappingPolicyInput) => {
		const res = await this.api.put(`/oidc/claim-mapping-policies/${id}`, policy);
		return res.data as OidcClaimMappingPolicyInput & { id: string };
	};

	deleteClaimMappingPolicy = async (id: string) => {
		await this.api.delete(`/oidc/claim-mapping-policies/${id}`);
	};

	previewClaimMappings = async (
		userId: string,
		scopes: string[],
		claimMappings: OidcClaimMappingDto[]
	) => {
		const res = await this.api.post('/oidc/claim-mapping-policies/preview', {
			userId,
			scopes,
			claimMappings
		});
		return res.data as OidcClaimMappingsPreview;
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

	listCustomScopes = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/oidc/custom-scopes', { params: options });
		return res.data as Paginated<CustomScope>;
	};

	listCustomScopeKeys = async () => {
		const res = await this.api.get('/oidc/custom-scopes/keys');
		return res.data as string[];
	};

	createCustomScope = async (scope: CustomScopeInput) => {
		const res = await this.api.post('/oidc/custom-scopes', scope);
		return res.data as CustomScope;
	};

	updateCustomScope = async (id: string, scope: CustomScopeInput) => {
		const res = await this.api.put(`/oidc/custom-scopes/${id}`, scope);
		return res.data as CustomScope;
	};

	deleteCustomScope = async (id: string) => {
		await this.api.delete(`/oidc/custom-scopes/${id}`);
	};
}

export default OidcClaimMappingPolicyService;

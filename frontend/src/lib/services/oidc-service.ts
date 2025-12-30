import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import type {
	AccessibleOidcClient,
	AuthorizeResponse,
	OidcClient,
	OidcClientCreate,
	OidcClientMetaData,
	OidcClientUpdate,
	OidcClientWithAllowedUserGroups,
	OidcClientWithAllowedUserGroupsCount,
	OidcDeviceCodeInfo
} from '$lib/types/oidc.type';
import type { ScimServiceProvider } from '$lib/types/scim.type';
import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
import APIService from './api-service';

class OidcService extends APIService {
	authorize = async (
		clientId: string,
		scope: string,
		callbackURL: string,
		nonce?: string,
		codeChallenge?: string,
		codeChallengeMethod?: string,
		reauthenticationToken?: string
	) => {
		const res = await this.api.post('/oidc/authorize', {
			scope,
			nonce,
			callbackURL,
			clientId,
			codeChallenge,
			codeChallengeMethod,
			reauthenticationToken
		});

		return res.data as AuthorizeResponse;
	};

	isAuthorizationRequired = async (clientId: string, scope: string) => {
		const res = await this.api.post('/oidc/authorization-required', {
			scope,
			clientId
		});

		return res.data.authorizationRequired as boolean;
	};

	listClients = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/oidc/clients', {
			params: options
		});
		return res.data as Paginated<OidcClientWithAllowedUserGroupsCount>;
	};

	createClient = async (client: OidcClientCreate) =>
		(await this.api.post('/oidc/clients', client)).data as OidcClient;

	removeClient = async (id: string) => {
		await this.api.delete(`/oidc/clients/${id}`);
	};

	getClient = async (id: string) =>
		(await this.api.get(`/oidc/clients/${id}`)).data as OidcClientWithAllowedUserGroups;

	getClientMetaData = async (id: string) =>
		(await this.api.get(`/oidc/clients/${id}/meta`)).data as OidcClientMetaData;

	updateClient = async (id: string, client: OidcClientUpdate) =>
		(await this.api.put(`/oidc/clients/${id}`, client)).data as OidcClient;

	updateClientLogo = async (client: OidcClient, image: File | null, light: boolean = true) => {
		const hasLogo = light ? client.hasLogo : client.hasDarkLogo;

		if (hasLogo && !image) {
			await this.removeClientLogo(client.id, light);
			return;
		}
		if (!hasLogo && !image) {
			return;
		}

		const formData = new FormData();
		formData.append('file', image!);

		await this.api.post(`/oidc/clients/${client.id}/logo`, formData, {
			params: { light }
		});
		cachedOidcClientLogo.bustCache(client.id, light);
	};

	removeClientLogo = async (id: string, light: boolean = true) => {
		await this.api.delete(`/oidc/clients/${id}/logo`, {
			params: { light }
		});
		cachedOidcClientLogo.bustCache(id, light);
	};

	createClientSecret = async (id: string) =>
		(await this.api.post(`/oidc/clients/${id}/secret`)).data.secret as string;

	updateAllowedUserGroups = async (id: string, userGroupIds: string[]) => {
		const res = await this.api.put(`/oidc/clients/${id}/allowed-user-groups`, { userGroupIds });
		return res.data as OidcClientWithAllowedUserGroups;
	};

	verifyDeviceCode = async (userCode: string) => {
		return await this.api.post(`/oidc/device/verify?code=${userCode}`);
	};

	getDeviceCodeInfo = async (userCode: string): Promise<OidcDeviceCodeInfo> => {
		const response = await this.api.get(`/oidc/device/info?code=${userCode}`);
		return response.data;
	};

	getClientPreview = async (id: string, userId: string, scopes: string) => {
		const response = await this.api.get(`/oidc/clients/${id}/preview/${userId}`, {
			params: { scopes }
		});
		return response.data;
	};

	listOwnAccessibleClients = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/oidc/users/me/clients', { params: options });
		return res.data as Paginated<AccessibleOidcClient>;
	};

	revokeOwnAuthorizedClient = async (clientId: string) => {
		await this.api.delete(`/oidc/users/me/authorized-clients/${clientId}`);
	};

	getScimResourceProvider = async (clientId: string) => {
		const res = await this.api.get(`/oidc/clients/${clientId}/scim-service-provider`);
		return res.data as ScimServiceProvider;
	};
}

export default OidcService;

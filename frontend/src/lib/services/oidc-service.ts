import type {
	AuthorizedOidcClient,
	AuthorizeResponse,
	OidcClient,
	OidcClientCreate,
	OidcClientMetaData,
	OidcClientWithAllowedUserGroups,
	OidcClientWithAllowedUserGroupsCount,
	OidcDeviceCodeInfo
} from '$lib/types/oidc.type';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
import APIService from './api-service';

class OidcService extends APIService {
	async authorize(
		clientId: string,
		scope: string,
		callbackURL: string,
		nonce?: string,
		codeChallenge?: string,
		codeChallengeMethod?: string,
		reauthenticationToken?: string
	) {
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
	}

	async isAuthorizationRequired(clientId: string, scope: string) {
		const res = await this.api.post('/oidc/authorization-required', {
			scope,
			clientId
		});

		return res.data.authorizationRequired as boolean;
	}

	async listClients(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/oidc/clients', {
			params: options
		});
		return res.data as Paginated<OidcClientWithAllowedUserGroupsCount>;
	}

	async createClient(client: OidcClientCreate) {
		return (await this.api.post('/oidc/clients', client)).data as OidcClient;
	}

	async removeClient(id: string) {
		await this.api.delete(`/oidc/clients/${id}`);
	}

	async getClient(id: string) {
		return (await this.api.get(`/oidc/clients/${id}`)).data as OidcClientWithAllowedUserGroups;
	}

	async getClientMetaData(id: string) {
		return (await this.api.get(`/oidc/clients/${id}/meta`)).data as OidcClientMetaData;
	}

	async updateClient(id: string, client: OidcClientCreate) {
		return (await this.api.put(`/oidc/clients/${id}`, client)).data as OidcClient;
	}

	async updateClientLogo(client: OidcClient, image: File | null) {
		if (client.hasLogo && !image) {
			await this.removeClientLogo(client.id);
			return;
		}
		if (!client.hasLogo && !image) {
			return;
		}

		const formData = new FormData();
		formData.append('file', image!);

		await this.api.post(`/oidc/clients/${client.id}/logo`, formData);
		cachedOidcClientLogo.bustCache(client.id);
	}

	async removeClientLogo(id: string) {
		await this.api.delete(`/oidc/clients/${id}/logo`);
		cachedOidcClientLogo.bustCache(id);
	}

	async createClientSecret(id: string) {
		return (await this.api.post(`/oidc/clients/${id}/secret`)).data.secret as string;
	}

	async updateAllowedUserGroups(id: string, userGroupIds: string[]) {
		const res = await this.api.put(`/oidc/clients/${id}/allowed-user-groups`, { userGroupIds });
		return res.data as OidcClientWithAllowedUserGroups;
	}

	async verifyDeviceCode(userCode: string) {
		return await this.api.post(`/oidc/device/verify?code=${userCode}`);
	}

	async getDeviceCodeInfo(userCode: string): Promise<OidcDeviceCodeInfo> {
		const response = await this.api.get(`/oidc/device/info?code=${userCode}`);
		return response.data;
	}

	async getClientPreview(id: string, userId: string, scopes: string) {
		const response = await this.api.get(`/oidc/clients/${id}/preview/${userId}`, {
			params: { scopes }
		});
		return response.data;
	}

	async listAuthorizedClients(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/oidc/users/me/clients', {
			params: options
		});
		return res.data as Paginated<AuthorizedOidcClient>;
	}

	async listAuthorizedClientsForUser(userId: string, options?: SearchPaginationSortRequest) {
		const res = await this.api.get(`/oidc/users/${userId}/clients`, {
			params: options
		});
		return res.data as Paginated<AuthorizedOidcClient>;
	}

	async revokeOwnAuthorizedClient(clientId: string) {
		await this.api.delete(`/oidc/users/me/clients/${clientId}`);
	}
}

export default OidcService;

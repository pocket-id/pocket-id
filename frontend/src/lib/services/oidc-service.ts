import type {
	AuthorizeResponse,
	OidcClient,
	OidcClientCreate,
	OidcClientWithAllowedUserGroups
} from '$lib/types/oidc.type';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import APIService from './api-service';

class OidcService extends APIService {
	async authorize(
		clientId: string,
		scope: string,
		callbackURL: string,
		nonce?: string,
		codeChallenge?: string,
		codeChallengeMethod?: string
	) {
		const res = await this.api.post('/oidc/authorize', {
			scope,
			nonce,
			callbackURL,
			clientId,
			codeChallenge,
			codeChallengeMethod
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
		return res.data as Paginated<OidcClient>;
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
	}

	async removeClientLogo(id: string) {
		await this.api.delete(`/oidc/clients/${id}/logo`);
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

	async deviceAuthorize(clientId: string, scope: string) {
		const params = new URLSearchParams();
		params.append('client_id', clientId);
		params.append('scope', scope);

		const response = await this.api.post('/oidc/device/authorize', params, {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded'
			}
		});
		return response.data;
	}
}

export default OidcService;

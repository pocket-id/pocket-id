import OIDCService from '$lib/services/oidc-service';
import type { ListRequestOptions } from '$lib/types/list-request.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const oidcService = new OIDCService();

	const appRequestOptions: ListRequestOptions = {
		pagination: {
			page: 1,
			limit: 20
		},
		sort: {
			column: 'lastUsedAt',
			direction: 'desc'
		},
		filters: {
			hasLaunchURL: [true]
		}
	};

	const authorizedClientRequestOptions: ListRequestOptions = {
		pagination: {
			page: 1,
			limit: 20
		},
		sort: {
			column: 'lastUsedAt',
			direction: 'desc'
		},
		filters: {
			hasLaunchURL: [false]
		}
	};

	const [clients, authorizedClientsWithoutLaunchURL] = await Promise.all([
		oidcService.listOwnAccessibleClients(appRequestOptions),
		oidcService.listOwnAuthorizedClients(authorizedClientRequestOptions)
	]);

	return {
		clients,
		appRequestOptions,
		authorizedClientsWithoutLaunchURL,
		authorizedClientRequestOptions
	};
};

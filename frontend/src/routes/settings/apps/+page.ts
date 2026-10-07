import OIDCService from '#lib/services/oidc-service.ts';
import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
import type { PageLoad } from './$types';
import { getMyAppsPreferences, myAppsSortOptions } from './my-apps-preferences.svelte.ts';

export const load: PageLoad = async () => {
	const oidcService = new OIDCService();
	const preferences = getMyAppsPreferences();

	const appRequestOptions: ListRequestOptions = {
		pagination: {
			page: 1,
			limit: preferences.paginationLimit
		},
		sort: { ...myAppsSortOptions[preferences.sort] },
		filters: {
			hasLaunchURL: [true]
		}
	};

	const authorizedClientRequestOptions: ListRequestOptions = {
		pagination: {
			page: 1,
			limit: preferences.paginationLimit
		},
		sort: { ...myAppsSortOptions[preferences.sort] },
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

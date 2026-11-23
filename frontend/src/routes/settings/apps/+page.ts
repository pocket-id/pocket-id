import OIDCService from '#lib/services/oidc-service.ts';
import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
import type { PageLoad } from './$types';
import { getMyAppsSort, myAppsSortOptions } from './my-apps-sort.svelte.ts';

// The server caps a page at 100, which also bounds where pinned apps are looked up
const options = (hasLaunchURL: boolean): ListRequestOptions => ({
	pagination: { page: 1, limit: 100 },
	sort: { ...myAppsSortOptions[getMyAppsSort()] },
	filters: { hasLaunchURL: [hasLaunchURL] }
});

export const load: PageLoad = async () => {
	const oidcService = new OIDCService();

	const appRequestOptions = options(true);
	const authorizedClientRequestOptions = options(false);

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

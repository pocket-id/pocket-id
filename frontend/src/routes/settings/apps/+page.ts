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
		}
	};

	const clients = await oidcService.listOwnAccessibleClients(appRequestOptions);

	return { clients, appRequestOptions };
};

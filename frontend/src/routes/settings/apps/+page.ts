import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
    const oidcService = new OIDCService();

    const appRequestOptions: SearchPaginationSortRequest = {
        pagination: {
            page: 1,
            limit: 20
        },
        sort: {
            column: 'lastUsedAt',
            direction: 'desc'
        }
    };

    const authorizedClients = await oidcService.listAuthorizedClients(appRequestOptions);

    return { authorizedClients, appRequestOptions };
};
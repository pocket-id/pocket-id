import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
    const oidcService = new OIDCService();

    const appRequestOptions: SearchPaginationSortRequest = {
        pagination: {
            page: 1,
            limit: 2
        },
        sort: {
            column: 'name',
            direction: 'asc'
        }
    };

    const apps = await oidcService.listAccessibleClients(appRequestOptions);

    return { apps, appRequestOptions };
};
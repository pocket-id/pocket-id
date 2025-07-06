import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
    const oidcService = new OIDCService();

    // this is only for testing we need to make routes to get this data by the users
    const appRequestOptions: SearchPaginationSortRequest = {
        sort: {
            column: 'name',
            direction: 'asc'
        }
    };

    const apps = await oidcService.listClients(appRequestOptions);

    return { apps, appRequestOptions };
};
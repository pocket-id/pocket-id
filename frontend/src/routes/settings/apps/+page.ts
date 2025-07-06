import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
    const oidcService = new OIDCService();

    const appRequestOptions: SearchPaginationSortRequest = {
        sort: {
            column: 'name',
            direction: 'asc'
        }
    };

    const apps = await oidcService.listClients(appRequestOptions);

    return { apps, appRequestOptions };
};
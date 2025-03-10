import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import AuditLogService from '$lib/services/audit-log-service';
import UserService from '$lib/services/user-service';
import OIDCService from '$lib/services/oidc-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, url }) => {
    const auditLogService = new AuditLogService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
    const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
    const oidcService = new OIDCService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

    // Get filters from URL parameters
    const userId = url.searchParams.get('userId') || undefined;
    const eventType = url.searchParams.get('eventType') || undefined;
    const clientId = url.searchParams.get('clientId') || undefined;
    
    // Create request options with default sorting and any filters
    const requestOptions: SearchPaginationSortRequest = {
        sort: {
            column: 'createdAt',
            direction: 'desc'
        },
        pagination: {
            page: 1,
            limit: 10
        }
    };
    
    // Only add filters if they exist
    if (userId || eventType || clientId) {
        requestOptions.filters = {};
        if (userId) requestOptions.filters.userId = userId;
        if (eventType) requestOptions.filters.event = eventType;
        if (clientId) requestOptions.filters.clientId = clientId;
    }

    // Get the audit logs
    const auditLogs = await auditLogService.listAllLogs(requestOptions);
    
    // Load supporting data for the filter dropdowns
    const userResponse = await userService.list();
    const clientResponse = await oidcService.listClients();
    
    // Prepare the data for the dropdowns
    const users = userResponse.data.map((user) => ({
        id: user.id,
        username: user.username || user.firstName + ' ' + user.lastName
    }));
    
    const clients = clientResponse.data.map((client) => ({
        id: client.id,
        name: client.name
    }));
    
    const eventTypes = [
        { value: 'SIGN_IN', label: 'Sign In' },
        { value: 'TOKEN_SIGN_IN', label: 'Token Sign In' },
        { value: 'CLIENT_AUTHORIZATION', label: 'Client Authorization' },
        { value: 'NEW_CLIENT_AUTHORIZATION', label: 'New Client Authorization' }
    ];

    return { 
        auditLogs,
        users,
        clients,
        eventTypes,
        filters: {
            userId,
            eventType,
            clientId
        }
    };
};

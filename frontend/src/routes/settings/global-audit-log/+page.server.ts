import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import AuditLogService from '$lib/services/audit-log-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const auditLogService = new AuditLogService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const requestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'createdAt',
			direction: 'desc'
		}
	};

	const auditLogs = await auditLogService.listAllLogs(requestOptions);

	const eventTypes = [
		{ value: 'SIGN_IN', label: 'Sign In' },
		{ value: 'TOKEN_SIGN_IN', label: 'Token Sign In' },
		{ value: 'CLIENT_AUTHORIZATION', label: 'Client Authorization' },
		{ value: 'NEW_CLIENT_AUTHORIZATION', label: 'New Client Authorization' }
	];

	return {
		auditLogs,
		eventTypes,
		requestOptions
	};
};

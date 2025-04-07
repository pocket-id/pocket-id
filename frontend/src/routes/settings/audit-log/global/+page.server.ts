import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import AuditLogService from '$lib/services/audit-log-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from '../../global-audit-log/$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const auditLogService = new AuditLogService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const requestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'createdAt',
			direction: 'desc'
		}
	};

	const auditLogs = await auditLogService.listAllLogs(requestOptions);

	return {
		auditLogs,
		requestOptions
	};
};

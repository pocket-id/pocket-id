import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import AuditLogService from '$lib/services/audit-log-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	const auditLogService = new AuditLogService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
	const auditLogsRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'createdAt',
			direction: 'desc'
		}
	};
	const auditLogs = await auditLogService.list(auditLogsRequestOptions);
	return { auditLogs, auditLogsRequestOptions };
};

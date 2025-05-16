import AuditLogService from '$lib/services/audit-log-service';
import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const auditLogService = new AuditLogService();
	const auditLogsRequestOptions: SearchPaginationSortRequest = {
		sort: {
			column: 'createdAt',
			direction: 'desc'
		}
	};
	const auditLogs = await auditLogService.list(auditLogsRequestOptions);
	return { auditLogs, auditLogsRequestOptions };
};

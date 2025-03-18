import type { AuditLog } from '$lib/types/audit-log.type';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import APIService from './api-service';

class AuditLogService extends APIService {
	async list(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/audit-logs', {
			params: options
		});
		return res.data as Paginated<AuditLog>;
	}

	async listAllLogs(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/audit-logs/all', {
			params: options
		});
		return res.data as Paginated<AuditLog>;
	}
}

export default AuditLogService;

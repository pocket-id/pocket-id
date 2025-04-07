import type { AuditLog, AuditLogFilter } from '$lib/types/audit-log.type';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import APIService from './api-service';

class AuditLogService extends APIService {
	async list(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/audit-logs', {
			params: options
		});
		return res.data as Paginated<AuditLog>;
	}

	async listAllLogs(options?: SearchPaginationSortRequest, filters?: AuditLogFilter) {
		const res = await this.api.get('/audit-logs/all', {
			params: {
				...options,
				filters
			}
		});
		return res.data as Paginated<AuditLog>;
	}

	async listClientNames() {
		const res = await this.api.get<string[]>('/audit-logs/filters/client-names');
		return res.data;
	}

	async listUsers() {
		const res = await this.api.get<Record<string, string>>('/audit-logs/filters/users');
		return res.data;
	}
}

export default AuditLogService;

import type { AuditLog, AuditLogFilter } from '$lib/types/audit-log.type';
import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import APIService from './api-service';

export default class AuditLogService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/audit-logs', { params: options });
		return res.data as Paginated<AuditLog>;
	};

	listAllLogs = async (options?: ListRequestOptions, filters?: AuditLogFilter) => {
		const res = await this.api.get('/audit-logs/all', { params: { ...options, filters } });
		return res.data as Paginated<AuditLog>;
	};

	listClientNames = async () => {
		const res = await this.api.get<string[]>('/audit-logs/filters/client-names');
		return res.data;
	};

	listUsers = async () => {
		const res = await this.api.get<Record<string, string>>('/audit-logs/filters/users');
		return res.data;
	};
}

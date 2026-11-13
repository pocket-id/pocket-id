import type { AuditLog, AuditLogDateRange } from '#lib/types/audit-log.type.ts';
import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
import APIService from './api-service';

export default class AuditLogService extends APIService {
	list = async (options?: ListRequestOptions, range?: AuditLogDateRange) => {
		const res = await this.api.get('/audit-logs', { params: { ...options, ...range } });
		return res.data as Paginated<AuditLog>;
	};

	listAllLogs = async (options?: ListRequestOptions, range?: AuditLogDateRange) => {
		const res = await this.api.get('/admin/audit-logs', { params: { ...options, ...range } });
		return res.data as Paginated<AuditLog>;
	};

	listClientNames = async () => {
		const res = await this.api.get<string[]>('/admin/audit-logs/filters/client-names');
		return res.data;
	};

	listUsers = async () => {
		const res = await this.api.get<Record<string, string>>('/admin/audit-logs/filters/users');
		return res.data;
	};
}

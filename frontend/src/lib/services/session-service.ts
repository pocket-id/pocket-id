import type { Session } from '#lib/types/session.type.ts';
import APIService from './api-service';
import { pathSegment } from '#lib/utils/path-segment-util.ts';

class SessionService extends APIService {
	list = async () => (await this.api.get('/sessions')).data as Session[];

	revoke = async (id: string) => {
		await this.api.delete(`/sessions/${pathSegment(id)}`);
	};

	rememberCurrent = async () => {
		await this.api.post('/sessions/current/remember');
	};

	revokeOthers = async () => {
		await this.api.delete('/sessions');
	};

	listForUser = async (userId: string) =>
		(await this.api.get(`/admin/users/${pathSegment(userId)}/sessions`)).data as Session[];

	revokeForUser = async (userId: string, id: string) => {
		await this.api.delete(`/admin/users/${pathSegment(userId)}/sessions/${pathSegment(id)}`);
	};

	revokeAllForUser = async (userId: string) => {
		await this.api.delete(`/admin/users/${pathSegment(userId)}/sessions`);
	};
}

export default SessionService;

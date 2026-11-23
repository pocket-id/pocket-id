import type { Session } from '#lib/types/session.type.ts';
import APIService from './api-service';

class SessionService extends APIService {
	list = async () => (await this.api.get('/sessions')).data as Session[];

	revoke = async (id: string) => {
		await this.api.delete(`/sessions/${id}`);
	};

	rememberCurrent = async () => {
		await this.api.post('/sessions/current/remember');
	};

	revokeOthers = async () => {
		await this.api.delete('/sessions');
	};

	listForUser = async (userId: string) =>
		(await this.api.get(`/admin/users/${userId}/sessions`)).data as Session[];

	revokeForUser = async (userId: string, id: string) => {
		await this.api.delete(`/admin/users/${userId}/sessions/${id}`);
	};

	revokeAllForUser = async (userId: string) => {
		await this.api.delete(`/admin/users/${userId}/sessions`);
	};
}

export default SessionService;

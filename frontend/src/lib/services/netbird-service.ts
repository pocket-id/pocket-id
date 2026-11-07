import type { NetBirdIdentity } from '#lib/types/netbird.type.ts';
import type { User } from '#lib/types/user.type.ts';
import APIService from './api-service';

export default class NetBirdService extends APIService {
	getIdentity = async () => (await this.api.get('/netbird/identity')).data as NetBirdIdentity;

	login = async (userId: string) =>
		(await this.api.post('/netbird/login', { userId })).data as User;
}

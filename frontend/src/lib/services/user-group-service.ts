import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import type {
	UserGroupCreate,
	UserGroupWithUserCount,
	UserGroupWithUsers
} from '$lib/types/user-group.type';
import APIService from './api-service';

export default class UserGroupService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/user-groups', { params: options });
		return res.data as Paginated<UserGroupWithUserCount>;
	};

	get = async (id: string) => {
		const res = await this.api.get(`/user-groups/${id}`);
		return res.data as UserGroupWithUsers;
	};

	create = async (user: UserGroupCreate) => {
		const res = await this.api.post('/user-groups', user);
		return res.data as UserGroupWithUsers;
	};

	update = async (id: string, user: UserGroupCreate) => {
		const res = await this.api.put(`/user-groups/${id}`, user);
		return res.data as UserGroupWithUsers;
	};

	remove = async (id: string) => {
		await this.api.delete(`/user-groups/${id}`);
	};

	updateUsers = async (id: string, userIds: string[]) => {
		const res = await this.api.put(`/user-groups/${id}/users`, { userIds });
		return res.data as UserGroupWithUsers;
	};
}

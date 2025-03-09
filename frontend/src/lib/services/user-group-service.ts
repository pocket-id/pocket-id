import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type {
	UserGroupCreate,
	UserGroupWithUserCount,
	UserGroupWithUsers
} from '$lib/types/user-group.type';
import APIService from './api-service';

export default class UserGroupService extends APIService {
	async list(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/user-groups', {
			params: options
		});
		return res.data as Paginated<UserGroupWithUserCount>;
	}

	async get(id: string, options?: SearchPaginationSortRequest) {
		const res = await this.api.get(`/user-groups/${id}`, {
			params: options
		});
		return res.data as UserGroupWithUsers;
	}

	async create(user: UserGroupCreate) {
		const res = await this.api.post('/user-groups', user);
		return res.data as UserGroupWithUsers;
	}

	async update(id: string, user: UserGroupCreate) {
		const res = await this.api.put(`/user-groups/${id}`, user);
		return res.data as UserGroupWithUsers;
	}

	async remove(id: string) {
		await this.api.delete(`/user-groups/${id}`);
	}

	async updateUsers(id: string, userIds: string[]) {
		const res = await this.api.put(`/user-groups/${id}/users`, { userIds });
		return res.data as UserGroupWithUsers;
	}
}

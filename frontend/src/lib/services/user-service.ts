import userStore from '$lib/stores/user-store';
import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
import type { UserGroup } from '$lib/types/user-group.type';
import type { User, UserCreate } from '$lib/types/user.type';
import { bustProfilePictureCache } from '$lib/utils/profile-picture-util';
import { get } from 'svelte/store';
import APIService from './api-service';

export default class UserService extends APIService {
	async list(options?: SearchPaginationSortRequest) {
		const res = await this.api.get('/users', {
			params: options
		});
		return res.data as Paginated<User>;
	}

	async get(id: string) {
		const res = await this.api.get(`/users/${id}`);
		return res.data as User;
	}

	async getCurrent() {
		const res = await this.api.get('/users/me');
		return res.data as User;
	}

	async create(user: UserCreate) {
		const res = await this.api.post('/users', user);
		return res.data as User;
	}

	async getUserGroups(userId: string) {
		const res = await this.api.get(`/users/${userId}/groups`);
		return res.data as UserGroup[];
	}

	async update(id: string, user: UserCreate) {
		const res = await this.api.put(`/users/${id}`, user);
		return res.data as User;
	}

	async updateCurrent(user: UserCreate) {
		const res = await this.api.put('/users/me', user);
		return res.data as User;
	}

	async remove(id: string) {
		await this.api.delete(`/users/${id}`);
	}

	async updateProfilePicture(userId: string, image: File) {
		const formData = new FormData();
		formData.append('file', image!);

		bustProfilePictureCache(userId);
		await this.api.put(`/users/${userId}/profile-picture`, formData);
	}

	async updateCurrentUsersProfilePicture(image: File) {
		const formData = new FormData();
		formData.append('file', image!);

		bustProfilePictureCache(get(userStore)!.id);
		await this.api.put('/users/me/profile-picture', formData);
	}

	async resetCurrentUserProfilePicture() {
		bustProfilePictureCache(get(userStore)!.id);
		await this.api.delete(`/users/me/profile-picture`);
	}

	async resetProfilePicture(userId: string) {
		bustProfilePictureCache(userId);
		await this.api.delete(`/users/${userId}/profile-picture`);
	}

	async createOneTimeAccessToken(expiresAt: Date, userId: string) {
		const res = await this.api.post(`/users/${userId}/one-time-access-token`, {
			userId,
			expiresAt
		});
		return res.data.token;
	}

	async exchangeOneTimeAccessToken(token: string) {
		const res = await this.api.post(`/one-time-access-token/${token}`);
		return res.data as User;
	}

	async requestOneTimeAccessEmail(email: string, redirectPath?: string) {
		await this.api.post('/one-time-access-email', { email, redirectPath });
	}

	async updateUserGroups(id: string, userGroupIds: string[]) {
		const res = await this.api.put(`/users/${id}/user-groups`, { userGroupIds });
		return res.data as User;
	}
}

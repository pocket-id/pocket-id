import userStore from '$lib/stores/user-store';
import type { ListRequestOptions, Paginated } from '$lib/types/list-request.type';
import type { SignupToken } from '$lib/types/signup-token.type';
import type { UserGroup } from '$lib/types/user-group.type';
import type { User, UserCreate, UserSignUp } from '$lib/types/user.type';
import { cachedProfilePicture } from '$lib/utils/cached-image-util';
import { get } from 'svelte/store';
import APIService from './api-service';
import { mapUser } from '$lib/utils/user-util';

export default class UserService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/users', { params: options });
		const data = res?.data?.data.map((d: User) => mapUser(d));
		return { ...res.data, data: data } as Paginated<User>;
	};

	get = async (id: string) => {
		const res = await this.api.get(`/users/${id}`);
		return mapUser(res.data as User);
	};

	getCurrent = async () => {
		const res = await this.api.get('/users/me');
		return mapUser(res.data as User);
	};

	create = async (user: UserCreate) => {
		const res = await this.api.post('/users', user);
		return mapUser(res.data as User);
	};

	getUserGroups = async (userId: string) => {
		const res = await this.api.get(`/users/${userId}/groups`);
		return res.data as UserGroup[];
	};

	update = async (id: string, user: UserCreate) => {
		const res = await this.api.put(`/users/${id}`, user);
		return mapUser(res.data as User);
	};

	updateCurrent = async (user: UserCreate) => {
		const res = await this.api.put('/users/me', user);
		return mapUser(res.data as User);
	};

	remove = async (id: string) => {
		await this.api.delete(`/users/${id}`);
	};

	updateProfilePicture = async (userId: string, image: File) => {
		const formData = new FormData();
		formData.append('file', image!);
		await this.api.put(`/users/${userId}/profile-picture`, formData);
		cachedProfilePicture.bustCache(userId);
	};

	updateCurrentUsersProfilePicture = async (image: File) => {
		const formData = new FormData();
		formData.append('file', image!);
		await this.api.put('/users/me/profile-picture', formData);
		cachedProfilePicture.bustCache(get(userStore)!.id);
	};

	resetCurrentUserProfilePicture = async () => {
		await this.api.delete(`/users/me/profile-picture`);
		cachedProfilePicture.bustCache(get(userStore)!.id);
	};

	resetProfilePicture = async (userId: string) => {
		await this.api.delete(`/users/${userId}/profile-picture`);
		cachedProfilePicture.bustCache(userId);
	};

	createOneTimeAccessToken = async (userId: string = 'me', ttl?: string | number) => {
		const res = await this.api.post(`/users/${userId}/one-time-access-token`, { userId, ttl });
		return res.data.token;
	};

	createSignupToken = async (
		ttl: string | number,
		usageLimit: number,
		userGroupIds: string[] = []
	) => {
		const res = await this.api.post(`/signup-tokens`, { ttl, usageLimit, userGroupIds });
		return res.data.token;
	};

	exchangeOneTimeAccessToken = async (token: string) => {
		const res = await this.api.post(`/one-time-access-token/${token}`);
		return mapUser(res.data as User);
	};

	requestOneTimeAccessEmailAsUnauthenticatedUser = async (email: string, redirectPath?: string) => {
		await this.api.post('/one-time-access-email', { email, redirectPath });
	};

	requestOneTimeAccessEmailAsAdmin = async (userId: string, ttl: string | number) => {
		await this.api.post(`/users/${userId}/one-time-access-email`, { ttl });
	};

	updateUserGroups = async (id: string, userGroupIds: string[]) => {
		const res = await this.api.put(`/users/${id}/user-groups`, { userGroupIds });
		return mapUser(res.data as User);
	};

	signup = async (data: UserSignUp) => {
		const res = await this.api.post(`/signup`, data);
		return mapUser(res.data as User);
	};

	signupInitialUser = async (data: UserSignUp) => {
		const res = await this.api.post(`/signup/setup`, data);
		return mapUser(res.data as User);
	};

	listSignupTokens = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/signup-tokens', { params: options });
		return res.data as Paginated<SignupToken>;
	};

	deleteSignupToken = async (tokenId: string) => {
		await this.api.delete(`/signup-tokens/${tokenId}`);
	};
}

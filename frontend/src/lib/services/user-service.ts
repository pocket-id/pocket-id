import { keepSignedIn } from '#lib/stores/keep-signed-in-store.svelte.ts';
import userStore from '#lib/stores/user-store.ts';
import type { ListRequestOptions, Paginated } from '#lib/types/list-request.type.ts';
import type { Passkey } from '#lib/types/passkey.type.ts';
import type { SignupToken } from '#lib/types/signup-token.type.ts';
import type { UserGroup } from '#lib/types/user-group.type.ts';
import type {
	AccountUpdate,
	EmailChange,
	User,
	UserCreate,
	UserSignUp
} from '#lib/types/user.type.ts';
import { cachedProfilePicture } from '#lib/utils/cached-asset-util.ts';
import { get } from 'svelte/store';
import APIService from './api-service';

export default class UserService extends APIService {
	list = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/admin/users', { params: options });
		return res.data as Paginated<User>;
	};

	get = async (id: string) => {
		const res = await this.api.get(`/admin/users/${id}`);
		return res.data as User;
	};

	getCurrent = async () => {
		const res = await this.api.get('/users/me');
		return res.data as User;
	};

	create = async (user: UserCreate) => {
		const res = await this.api.post('/admin/users', user);
		return res.data as User;
	};

	getUserGroups = async (userId: string) => {
		const res = await this.api.get(`/admin/users/${userId}/groups`);
		return res.data as UserGroup[];
	};

	listUserPasskeys = async (userId: string) => {
		const res = await this.api.get(`/admin/users/${userId}/webauthn-credentials`);
		return res.data as Passkey[];
	};

	update = async (id: string, user: Partial<UserCreate>) => {
		const res = await this.api.patch(`/admin/users/${id}`, user);
		return res.data as User;
	};

	updateCurrent = async (user: Partial<AccountUpdate>) => {
		const res = await this.api.patch('/users/me', user);
		return res.data as User;
	};

	remove = async (id: string) => {
		await this.api.delete(`/admin/users/${id}`);
	};

	removeUserPasskey = async (userId: string, passkeyId: string) => {
		await this.api.delete(`/admin/users/${userId}/webauthn-credentials/${passkeyId}`);
	};

	updateProfilePicture = async (userId: string, image: File) => {
		const formData = new FormData();
		formData.append('file', image!);
		await this.api.put(`/admin/users/${userId}/profile-picture`, formData);
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
		await this.api.delete(`/admin/users/${userId}/profile-picture`);
		cachedProfilePicture.bustCache(userId);
	};

	createOneTimeAccessToken = async (userId: string, ttl?: string | number) => {
		const res = await this.api.post(`/admin/users/${userId}/one-time-access-token`, { ttl });
		return res.data.token;
	};

	createSignupToken = async (
		ttl: string | number,
		usageLimit: number,
		userGroupIds: string[] = []
	) => {
		const res = await this.api.post(`/admin/signup-tokens`, { ttl, usageLimit, userGroupIds });
		return res.data.token;
	};

	exchangeOneTimeAccessToken = async (token: string) => {
		const res = await this.api.post(
			'/one-time-access-token',
			{ token },
			{
				params: { remember: keepSignedIn.current }
			}
		);
		return res.data as User;
	};

	requestOneTimeAccessEmailAsUnauthenticatedUser = async (email: string, redirectPath?: string) => {
		await this.api.post('/one-time-access-email', { email, redirectPath });
	};

	requestOneTimeAccessEmailAsAdmin = async (userId: string, ttl: string | number) => {
		await this.api.post(`/admin/users/${userId}/one-time-access-email`, { ttl });
	};

	updateUserGroups = async (id: string, userGroupIds: string[]) => {
		const res = await this.api.put(`/admin/users/${id}/user-groups`, { userGroupIds });
		return res.data as User;
	};

	signup = async (data: UserSignUp) => {
		const res = await this.api.post(`/signup`, data);
		return res.data as User;
	};

	signupInitialUser = async (data: UserSignUp) => {
		const res = await this.api.post(`/signup/setup`, data);
		return res.data as User;
	};

	checkInitialUserSetupAvailable = async () => {
		await this.api.get('/signup/setup');
	};

	listSignupTokens = async (options?: ListRequestOptions) => {
		const res = await this.api.get('/admin/signup-tokens', { params: options });
		return res.data as Paginated<SignupToken>;
	};

	deleteSignupToken = async (tokenId: string) => {
		await this.api.delete(`/admin/signup-tokens/${tokenId}`);
	};

	sendEmailVerification = async () => {
		const res = await this.api.post('/users/me/send-email-verification');
		return res.data as User;
	};

	verifyEmail = async (token: string) => {
		const res = await this.api.post('/users/me/verify-email', { token });
		return res.data as User;
	};

	getPendingEmailChange = async () => {
		const res = await this.api.get('/users/me/email-change');
		return res.status === 204 ? null : (res.data as EmailChange);
	};

	cancelEmailChange = async () => {
		await this.api.delete('/users/me/email-change');
	};
}

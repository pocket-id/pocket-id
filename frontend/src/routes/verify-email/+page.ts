import UserService from '$lib/services/user-service';
import { getAxiosErrorMessage } from '$lib/utils/error-util';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const userService = new UserService();
	const token = url.searchParams.get('token');

	const searchParams = new URLSearchParams();

	await userService
		.verifyEmail(token!)
		.then(() => {
			searchParams.set('emailVerificationState', 'success');
		})
		.catch((e) => {
			searchParams.set('emailVerificationState', getAxiosErrorMessage(e));
		});

	return redirect(302, '/settings/account?' + searchParams.toString());
};

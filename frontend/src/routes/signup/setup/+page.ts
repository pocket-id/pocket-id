import { error } from '@sveltejs/kit';
import UserService from '$lib/services/user-service';
import { AxiosError } from 'axios';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();

	try {
		await userService.checkInitialUserSetupAvailable();
	} catch (e) {
		if (e instanceof AxiosError && e.response?.status === 404) {
			error(404, 'Not found');
		}

		throw e;
	}
};

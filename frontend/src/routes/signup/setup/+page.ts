import { error } from '@sveltejs/kit';
import { m } from '#lib/paraglide/messages.js';
import UserService from '#lib/services/user-service.ts';
import { AxiosError } from 'axios';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const userService = new UserService();

	try {
		await userService.checkInitialUserSetupAvailable();
	} catch (e) {
		if (e instanceof AxiosError && e.response?.status === 404) {
			error(404, m.page_not_found());
		}

		throw e;
	}
};

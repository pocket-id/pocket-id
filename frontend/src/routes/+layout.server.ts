import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import { m } from '$lib/paraglide/messages';
import AppConfigService from '$lib/services/app-config-service';
import UserService from '$lib/services/user-service';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies }) => {
	const userService = new UserService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));
	const appConfigService = new AppConfigService(cookies.get(ACCESS_TOKEN_COOKIE_NAME));

	const user = await userService
		.getCurrent()
		.then((user) => user)
		.catch(() => null);

	const appConfig = await appConfigService
		.list()
		.then((config) => config)
		.catch((e) => {
			console.error(
				m.failed_to_get_application_configuration({ message: e.response?.data.error || e.message })
			);
			return null;
		});
	return {
		user,
		appConfig
	};
};

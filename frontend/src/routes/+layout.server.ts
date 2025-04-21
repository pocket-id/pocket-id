import { ACCESS_TOKEN_COOKIE_NAME } from '$lib/constants';
import AppConfigService from '$lib/services/app-config-service';
import UserService from '$lib/services/user-service';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies }) => {
	const accessToken = cookies.get(ACCESS_TOKEN_COOKIE_NAME);
	const userService = new UserService(accessToken);
	const appConfigService = new AppConfigService(accessToken);

	const userPromise = userService.getCurrent().catch(() => null);
	
	const appConfigPromise = appConfigService.list().catch((e) => {
		console.error(
			`Failed to get application configuration: ${e.response?.data.error || e.message}`
		);
		return null;
	});

	const [user, appConfig] = await Promise.all([userPromise, appConfigPromise]);

	return {
		user,
		appConfig
	};
};

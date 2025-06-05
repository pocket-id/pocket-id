import AppConfigService from '$lib/services/app-config-service';
import UserService from '$lib/services/user-service';
import type { LayoutLoad } from './$types';

export const ssr = false;

export const load: LayoutLoad = async () => {
	const userService = new UserService();
	const appConfigService = new AppConfigService();

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

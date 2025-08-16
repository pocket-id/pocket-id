import AppConfigService from '$lib/services/app-config-service';
import UserGroupService from '$lib/services/user-group-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const appConfigService = new AppConfigService();
	const userGroupService = new UserGroupService();

	const [appConfig, userGroups] = await Promise.all([
		appConfigService.list(true),
		userGroupService.list({ pagination: { limit: 1000, page: 1 } }) // Cargar todos los grupos
	]);

	return {
		appConfig,
		userGroups: userGroups.data
	};
};

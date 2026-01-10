import AppConfigService from '$lib/services/app-config-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const appConfigService = new AppConfigService();

	const appConfigData = await appConfigService.list(true);

	return {
		emailsVerifiedPerDefault: appConfigData.emailsVerified
	};
};

import AppConfigService from '#lib/services/app-config-service.ts';
import type { PageLoad } from './$types';
import { countUsersByScope } from './user-scopes';

export const load: PageLoad = async () => {
	const [appConfigData, counts] = await Promise.all([
		new AppConfigService().list(true),
		countUsersByScope()
	]);

	return {
		emailsVerifiedPerDefault: appConfigData.emailsVerified,
		counts
	};
};

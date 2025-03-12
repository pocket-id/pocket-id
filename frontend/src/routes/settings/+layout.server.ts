import { version as currentVersion } from '$app/environment';
import { env } from '$env/dynamic/private';
import AppConfigService from '$lib/services/app-config-service';
import type { AppVersionInformation } from '$lib/types/application-configuration';
import type { LayoutServerLoad } from './$types';

let versionInformation: AppVersionInformation;
let versionInformationLastUpdated: number;

export const load: LayoutServerLoad = async () => {
	if (env.UPDATE_CHECK_DISABLED === 'true') {
		return {
			versionInformation: {
				currentVersion: currentVersion
			} satisfies AppVersionInformation
		};
	}

	const appConfigService = new AppConfigService();

	// Cache the version information for 3 hours
	const cacheExpired =
		versionInformationLastUpdated &&
		Date.now() - versionInformationLastUpdated > 1000 * 60 * 60 * 3;

	if (!versionInformation || cacheExpired) {
		versionInformation = await appConfigService.getVersionInformation();
		if (versionInformation.newestVersion == null) {
			console.error('Failed to fetch version information. Trying again in 3 hours.');
		}
		versionInformationLastUpdated = Date.now();
	}

	return {
		versionInformation
	};
};

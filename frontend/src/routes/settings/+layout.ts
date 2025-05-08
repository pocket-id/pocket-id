import versionService from '$lib/services/version-service';
import type { AppVersionInformation } from '$lib/types/application-configuration';
import type { LayoutLoad } from './$types';

export const prerender = false;

export const load: LayoutLoad = async () => {
	const versionInformation: AppVersionInformation = {
		currentVersion: versionService.getCurrentVersion(),
		newestVersion: await versionService.getNewestVersion(),
		isUpToDate: await versionService.isUpToDate()
	};

	return {
		versionInformation
	};
};

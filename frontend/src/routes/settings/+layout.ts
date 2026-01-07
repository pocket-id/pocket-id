import VersionService from '$lib/services/version-service';
import type { AppVersionInformation } from '$lib/types/application-configuration.type';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
	const versionService = new VersionService();
	const currentVersion = versionService.getCurrentVersion();

	let newestVersion = null;
	let isUpToDate = true;
	try {
		newestVersion = await versionService.getNewestVersion();
		isUpToDate = newestVersion === currentVersion;
	} catch {}

	const versionInformation: AppVersionInformation = {
		currentVersion: versionService.getCurrentVersion(),
		newestVersion,
		isUpToDate
	};

	return {
		versionInformation
	};
};

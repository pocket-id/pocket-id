import StorageService from '$lib/services/storage-service';
import VersionService from '$lib/services/version-service';
import type { AppVersionInformation } from '$lib/types/application-configuration.type';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
	const versionService = new VersionService();
	const storageService = new StorageService();
	const currentVersion = versionService.getCurrentVersion();

	let newestVersion = null;
	let isUpToDate: boolean;
	try {
		newestVersion = await versionService.getNewestVersion();
		// If newestVersion is empty, it means the check is disabled or failed.
		// In this case, we assume the version is up to date.
		isUpToDate = newestVersion === '' || newestVersion === currentVersion;
	} catch {
		// If the request fails, assume up-to-date to avoid showing a warning.
		isUpToDate = true;
	}

	const versionInformation: AppVersionInformation = {
		currentVersion: versionService.getCurrentVersion(),
		newestVersion,
		isUpToDate
	};

	// If the request fails, don't show the warning to avoid a false positive.
	const sqliteStorageWarning = await storageService.getSqliteStorageWarning().catch(() => false);

	return {
		versionInformation,
		sqliteStorageWarning
	};
};

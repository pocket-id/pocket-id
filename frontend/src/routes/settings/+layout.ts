import StorageService from '#lib/services/storage-service.ts';
import VersionService from '#lib/services/version-service.ts';
import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async () => {
	const versionService = new VersionService();
	const storageService = new StorageService();
	const currentVersion = versionService.getCurrentVersion();

	const [newestVersion, sqliteStorageWarning] = await Promise.all([
		versionService.getNewestVersion().catch(() => null),
		storageService.getSqliteStorageWarning().catch(() => false)
	]);

	// If newestVersion is empty, it means the check is disabled or failed.
	// In this case, we assume the version is up to date.
	const isUpToDate =
		newestVersion === null || newestVersion === '' || newestVersion === currentVersion;

	const versionInformation: AppVersionInformation = {
		currentVersion: versionService.getCurrentVersion(),
		newestVersion,
		isUpToDate
	};

	return {
		versionInformation,
		sqliteStorageWarning
	};
};

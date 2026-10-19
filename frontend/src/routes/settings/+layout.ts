import FeatureService from '#lib/services/feature-service.ts';
import StorageService from '#lib/services/storage-service.ts';
import VersionService from '#lib/services/version-service.ts';
import WebAuthnService from '#lib/services/webauthn-service.ts';
import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ url, parent, untrack, depends }) => {
	depends('app:features');

	const versionService = new VersionService();
	const storageService = new StorageService();
	const webauthnService = new WebAuthnService();

	const currentVersion = versionService.getCurrentVersion();

	const [newestVersion, sqliteStorageWarning, passkeys, features] = await Promise.all([
		versionService.getNewestVersion().catch(() => null),
		storageService.getSqliteStorageWarning().catch(() => false),
		webauthnService.listCredentials(),
		// A tracked parent() would rerun this load on every navigation, because the root load reads the pathname
		untrack(() => parent()).then(({ user }) => (user?.isAdmin ? new FeatureService().list() : null))
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

	const skipPasskeySetup =
		parseInt(localStorage.getItem('skip-passkey-setup-until') ?? '0') > Date.now();

	if (!skipPasskeySetup && passkeys.length === 0 && url.pathname !== '/signup/add-passkey') {
		redirect(303, '/signup/add-passkey');
	}

	return {
		versionInformation,
		sqliteStorageWarning,
		passkeys,
		features
	};
};

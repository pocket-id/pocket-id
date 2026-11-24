import FeatureService from '#lib/services/feature-service.ts';
import VersionService from '#lib/services/version-service.ts';
import WarningService from '#lib/services/warning-service.ts';
import WebAuthnService from '#lib/services/webauthn-service.ts';
import type { AppVersionInformation } from '#lib/types/application-configuration.type.ts';
import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ url, depends, parent }) => {
	depends('app:features');

	const versionService = new VersionService();
	const webauthnService = new WebAuthnService();

	const currentVersion = versionService.getCurrentVersion();

	const { user } = await parent();

	const [newestVersion, warnings, passkeys, features] = await Promise.all([
		versionService.getNewestVersion().catch(() => null),
		// Only admins may read the warnings, and a failed check shows none
		user?.isAdmin ? new WarningService().list().catch(() => []) : [],
		webauthnService.listCredentials(),
		new FeatureService().list()
	]);

	// If newestVersion is empty, it means the check is disabled or failed.
	// In this case, we assume the version is up to date.
	// Next builds (e.g. "v2.19.1-next") are built ahead of the latest release, so they never need an update.
	const isUpToDate =
		newestVersion === null ||
		newestVersion === '' ||
		newestVersion === currentVersion ||
		currentVersion.endsWith('-next');

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
		warnings,
		passkeys,
		features
	};
};

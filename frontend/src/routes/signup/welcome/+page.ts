import FeatureService from '#lib/services/feature-service.ts';
import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Only an admin can turn features on, and an environment-managed configuration cannot be saved from here
export const load: PageLoad = async ({ parent }) => {
	const { user, appConfig } = await parent();
	if (!user?.isAdmin || appConfig?.uiConfigDisabled) redirect(303, '/settings');

	return { features: await new FeatureService().list() };
};

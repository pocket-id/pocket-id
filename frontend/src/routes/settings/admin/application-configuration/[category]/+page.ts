import { m } from '#lib/paraglide/messages.js';
import AppConfigService from '#lib/services/app-config-service.ts';
import type { OidcDiscoveryConfiguration } from '#lib/types/oidc.type.ts';
import { error } from '@sveltejs/kit';
import { categories, type Category } from '../categories.ts';
import type { PageLoad } from './$types';

// Every tab loads the configuration again, so a tab never starts from values another tab has saved since
export const load: PageLoad = async ({ params, fetch }) => {
	const category = params.category as Category;
	if (!categories.includes(category)) error(404, m.page_not_found());

	const [{ appConfig, envOverrides }, oidcConfiguration] = await Promise.all([
		new AppConfigService().listWithEnvOverrides(),
		category === 'oidc'
			? fetch('/.well-known/openid-configuration').then(
					(res) => res.json() as Promise<OidcDiscoveryConfiguration>
				)
			: undefined
	]);

	return { category, appConfig, envOverrides, oidcConfiguration };
};

import type { RequestHandler } from './$types';
import type { AppConfigRawResponse } from '$lib/types/application-configuration';

export const prerender = true;

export const GET: RequestHandler = async ({ fetch }) => {
	const response = await fetch(`/api/application-configuration`);
	const data: AppConfigRawResponse = await response.json();

	const appNameConfig = data.find((config) => config.key === 'appName');
	const appName = appNameConfig?.value || 'Pocket ID';

	const manifest = {
		name: appName,
		icons: [
			{
				src: '/api/application-configuration/pwa-icon',
				sizes: '512x512',
				type: 'image/png',
				purpose: 'any'
			}
		],
		display: 'browser'
	};

	return new Response(JSON.stringify(manifest), {
		headers: {
			'Content-Type': 'application/manifest+json'
		}
	});
};

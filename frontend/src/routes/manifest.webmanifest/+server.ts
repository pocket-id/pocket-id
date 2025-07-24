import type { RequestHandler } from './$types';
export const prerender = true; // this should just prerender manifest.json and not treat it as a server route

export const GET: RequestHandler = async () => {
	const manifest = {
		name: 'PocketID',
		icons: [
			{
				src: '/api/application-configuration/logo',
				sizes: 'any'
			}
		],
		display: 'browser',
		background_color: '#000000',
		theme_color: '#000000'
	};

	return new Response(JSON.stringify(manifest), {
		headers: {
			'Content-Type': 'application/manifest+json'
		}
	});
};

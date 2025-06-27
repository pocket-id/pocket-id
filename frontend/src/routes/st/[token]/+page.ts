import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Alias for /signup?token=...
export const load: PageLoad = async ({ url, params }) => {
	const targetPath = '/signup';

	const searchParams = new URLSearchParams();
	searchParams.set('token', params.token);

	if (url.searchParams.has('redirect')) {
		searchParams.set('redirect', url.searchParams.get('redirect')!);
	}

	return redirect(307, `${targetPath}?${searchParams.toString()}`);
};

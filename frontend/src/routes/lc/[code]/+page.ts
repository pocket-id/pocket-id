import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Alias for /login/alternative/code?code=...
export const load: PageLoad = async ({ url, params }) => {
	const targetPath = '/login/alternative/code';

	const searchParams = new URLSearchParams();
	searchParams.set('code', params.code);

	if (url.searchParams.has('redirect')) {
		searchParams.set('redirect', url.searchParams.get('redirect')!);
	}

	return redirect(307, `${targetPath}?${searchParams.toString()}`);
}

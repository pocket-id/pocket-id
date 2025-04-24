import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Alias for /login/alternative/code
export const load: PageLoad = async ({ url }) => {
	let targetPath = '/login/alternative/code';
	if (url.searchParams.has('redirect')) {
		targetPath += `?redirect=${encodeURIComponent(url.searchParams.get('redirect')!)}`;
	}
	return redirect(307, targetPath);
}

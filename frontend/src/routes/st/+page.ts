import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Alias for /signup
export const load: PageLoad = async ({ url }) => {
	let targetPath = '/signup';
	if (url.searchParams.has('redirect')) {
		targetPath += `?redirect=${encodeURIComponent(url.searchParams.get('redirect')!)}`;
	}
	if (url.searchParams.has('token')) {
		const separator = targetPath.includes('?') ? '&' : '?';
		targetPath += `${separator}token=${encodeURIComponent(url.searchParams.get('token')!)}`;
	}
	return redirect(307, targetPath);
};

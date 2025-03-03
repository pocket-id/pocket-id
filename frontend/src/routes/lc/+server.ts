import { redirect } from '@sveltejs/kit';

// Alias for /login/alternative/code
export function GET({ url }) {
	let targetPath = '/login/alternative/code';
	if (url.searchParams.has('redirect')) {
		targetPath += `?redirect=${encodeURIComponent(url.searchParams.get('redirect')!)}`;
	}
	return redirect(307, targetPath);
}

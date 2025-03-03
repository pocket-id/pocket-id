import { redirect } from '@sveltejs/kit';

// Alias for /login/alternative/code?code=...
export function GET({ url, params }) {
	const targetPath = '/login/alternative/code';

	const searchParams = new URLSearchParams();
	searchParams.set('code', params.code);

	if (url.searchParams.has('redirect')) {
		searchParams.set('redirect', url.searchParams.get('redirect')!);
	}

	return redirect(307, `${targetPath}?${searchParams.toString()}`);
}

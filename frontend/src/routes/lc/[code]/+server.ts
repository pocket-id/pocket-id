import { env } from '$env/dynamic/public';
import { redirect } from '@sveltejs/kit';

export function GET({ url, params }) {
	const targetUrl = new URL('/login/alternative/code', env.PUBLIC_APP_URL);

	targetUrl.searchParams.set('code', params.code);

	if (url.searchParams.has('redirect')) {
		targetUrl.searchParams.set('redirect', url.searchParams.get('redirect')!);
	}
	return redirect(307, targetUrl.toString());
}

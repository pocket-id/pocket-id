import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	return {
		code: url.searchParams.get('code'),
		redirect: url.searchParams.get('redirect') || '/settings'
	};
};

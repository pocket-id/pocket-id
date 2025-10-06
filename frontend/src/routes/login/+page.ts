import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	return {
		redirect: url.searchParams.get('redirect') || '/settings'
	};
};

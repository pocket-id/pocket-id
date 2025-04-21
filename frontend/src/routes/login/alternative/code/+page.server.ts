import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	return {
		code: url.searchParams.get('code'),
		redirect: url.searchParams.get('redirect') || '/settings'
	};
};

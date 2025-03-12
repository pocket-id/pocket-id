import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	return {
		redirect: url.searchParams.get('redirect') || undefined
	};
};

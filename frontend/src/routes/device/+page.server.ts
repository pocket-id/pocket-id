import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const code = url.searchParams.get('code');

	return {
		code
	};
};

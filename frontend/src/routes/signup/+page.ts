import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	return {
		token: url.searchParams.get('token') || undefined
	};
};

import { m } from '#lib/paraglide/messages.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
	const error = url.searchParams.get('error') ?? m.an_unknown_error_occurred();

	return {
		error
	};
};

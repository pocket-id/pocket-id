import ApisService from '$lib/services/apis-service';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params }) => {
	const api = await new ApisService().get(params.id);
	return { api };
};

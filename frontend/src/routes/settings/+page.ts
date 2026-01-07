import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import appConfig from '$lib/stores/application-configuration-store';
import { get } from 'svelte/store';

export const load: PageLoad = async () => {
	throw redirect(307, get(appConfig).homePageUrl);
};

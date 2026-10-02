import appConfig from '#lib/stores/application-configuration-store.ts';
import { redirect } from '@sveltejs/kit';
import { get } from 'svelte/store';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	redirect(307, get(appConfig)?.homePageUrl ?? '/');
};

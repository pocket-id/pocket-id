import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Alias for /signup/setup
export const load: PageLoad = async () => redirect(307, '/signup/setup');

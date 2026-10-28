import { m } from '#lib/paraglide/messages.js';
import { error } from '@sveltejs/kit';

// Unknown paths under /settings would otherwise fall through to the root error page and lose the shell
export const load = () => error(404, m.page_not_found());

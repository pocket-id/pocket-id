import SessionService from '#lib/services/session-service.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => ({ sessions: await new SessionService().list() });

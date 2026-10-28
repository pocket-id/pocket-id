import UserGroupService from '#lib/services/user-group-service.ts';
import type { PageLoad } from './$types';

// Known before the first render, so a list without groups does not flash the full layout before its first-run state
export const load: PageLoad = async () => {
	const groups = await new UserGroupService().list({ pagination: { page: 1, limit: 1 } });
	return { empty: groups.pagination.totalItems === 0 };
};

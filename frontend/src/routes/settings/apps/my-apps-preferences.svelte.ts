import type { SortRequest } from '#lib/types/list-request.type.ts';
import { PersistedState } from 'runed';

export const myAppsSortOptions = {
	recentlyUsed: { column: 'lastUsedAt', direction: 'desc' },
	nameAsc: { column: 'name', direction: 'asc' },
	nameDesc: { column: 'name', direction: 'desc' }
} satisfies Record<string, SortRequest>;

export type MyAppsSort = keyof typeof myAppsSortOptions;

export const myAppsPageSizes = [20, 50, 100];

type MyAppsPreferences = {
	sort: MyAppsSort;
	paginationLimit: number;
};

const defaultPreferences: MyAppsPreferences = {
	sort: 'recentlyUsed',
	paginationLimit: myAppsPageSizes[0]
};

export const myAppsPreferences = new PersistedState<MyAppsPreferences>(
	'my-apps-preferences',
	defaultPreferences
);

// Stored values can be stale or edited by hand, so unknown ones fall back to the defaults
export function getMyAppsPreferences(): MyAppsPreferences {
	const { sort, paginationLimit } = myAppsPreferences.current;
	return {
		sort: sort in myAppsSortOptions ? sort : defaultPreferences.sort,
		paginationLimit: myAppsPageSizes.includes(paginationLimit)
			? paginationLimit
			: defaultPreferences.paginationLimit
	};
}

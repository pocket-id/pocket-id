import type { SortRequest } from '#lib/types/list-request.type.ts';
import { PersistedState } from 'runed';

export const myAppsSortOptions = {
	nameAsc: { column: 'name', direction: 'asc' },
	nameDesc: { column: 'name', direction: 'desc' },
	recentlyUsed: { column: 'lastUsedAt', direction: 'desc' }
} satisfies Record<string, SortRequest>;

export type MyAppsSort = keyof typeof myAppsSortOptions;

export const myAppsSort = new PersistedState<MyAppsSort>('my-apps-sort', 'nameAsc');

// Stored values can be stale or edited by hand, so an unknown one falls back to A–Z
export function getMyAppsSort(): MyAppsSort {
	return myAppsSort.current in myAppsSortOptions ? myAppsSort.current : 'nameAsc';
}

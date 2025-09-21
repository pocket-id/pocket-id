import type { Row, Column, FilterFn, ColumnFiltersState, VisibilityState } from '@tanstack/table-core';
import type { Snippet } from 'svelte';

export type ColumnSpec<T> = {
	accessorKey?: keyof T & string;
	accessorFn?: (row: T) => any;
	id?: string;
	title: string;
	hidden?: boolean;
	sortable?: boolean;
	cell?: Snippet<[{ row: Row<T>; item: T; value: unknown }]>;
	header?: Snippet<[{ column: Column<T>; title: string; class?: string }]>;
	class?: string;
	filterFn?: FilterFn<T>;
};

// Compact persisted prefs to reduce JSON size
export type CompactTablePrefs = {
	// v: list of hidden column ids (only the exceptions)
	v?: string[];
	// f: filters as [id, value] tuples
	f?: [string, unknown][];
	// g: global filter string
	g?: string;
	// l: page size (limit)
	l?: number;
};

export function encodeHidden(visibility: VisibilityState): string[] {
	const hidden: string[] = [];
	for (const [id, visible] of Object.entries(visibility)) {
		if (visible === false) hidden.push(id);
	}
	return hidden;
}

export function applyHiddenPatch(target: VisibilityState, hidden?: string[]) {
	if (!hidden?.length) return;
	for (const id of hidden) {
		target[id] = false;
	}
}

export function encodeFilters(filters: ColumnFiltersState): [string, unknown][] {
	return (filters ?? []).map((f) => [f.id, (f as any).value] as [string, unknown]);
}

export function decodeFilters(pairs?: [string, unknown][]): ColumnFiltersState {
	if (!pairs?.length) return [];
	return pairs.map(([id, value]) => ({ id, value }));
}

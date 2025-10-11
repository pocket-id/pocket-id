import type { Component, Snippet } from 'svelte';

export type AdvancedTableColumn<T extends Record<string, any>> = {
	label: string;
	column?: keyof T & string;
    key?: string;
	value?: (item: T) => string | number | boolean | undefined;
	cell?: Snippet<[{ item: T }]>;
	sortable?: boolean;
	filterableValues?: {
		label: string;
		value: string | boolean;
		icon?: Component;
	}[];
    hidden?: boolean;
};
export type CreateAdvancedTableActions<T extends Record<string, any>> = (item: T) => AdvancedTableAction<T>[];

export type AdvancedTableAction<T> = {
	label: string;
	icon?: Component;
	variant?: 'primary' | 'secondary' | 'danger' | 'outline' | 'ghost';
	onClick: (item: T) => void;
	visible?: boolean;
	disabled?: boolean;
};

<script lang="ts" generics="TData">
	import XIcon from '@lucide/svelte/icons/x';
	import type { Table } from '@tanstack/table-core';
	import { DataTableFacetedFilter, DataTableViewOptions } from './index.js';
	import Button from '$lib/components/ui/button/button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { userRoleFilters, disabledFilters } from './data.js';
	import { debounced } from '$lib/utils/debounce-util.js';
	import { m } from '$lib/paraglide/messages';

	let {
		table,
		selectedIds = [],
		selectionDisabled = false,
		onRemoveSelected
	}: {
		table: Table<TData>;
		selectedIds?: string[];
		selectionDisabled?: boolean;
		onRemoveSelected?: (ids: string[]) => void;
	} = $props();

	const isFiltered = $derived(
		table.getState().columnFilters.length > 0 || !!table.getState().globalFilter
	);
	const disabledUserColumn = $derived(table.getColumn('disabled'));
	const userRoleColumn = $derived(table.getColumn('isAdmin'));

	const debouncedSetGlobal = debounced((v: string) => table.setGlobalFilter(v), 300);
	const hasSelection = $derived(!selectionDisabled && (selectedIds?.length ?? 0) > 0);
</script>

<div class="flex items-center justify-between">
	<div class="flex flex-1 items-center space-x-2">
		<Input
			placeholder={m.search()}
			value={(table.getState().globalFilter as string) ?? ''}
			oninput={(e) => debouncedSetGlobal(e.currentTarget.value)}
			onchange={(e) => table.setGlobalFilter(e.currentTarget.value)}
			onkeydown={(e) => {
				if (e.key === 'Enter') table.setGlobalFilter((e.currentTarget as HTMLInputElement).value);
			}}
			class="h-8 w-[150px] lg:w-[250px]"
		/>

		{#if disabledUserColumn}
			<DataTableFacetedFilter
				column={disabledUserColumn}
				title={m.status()}
				options={disabledFilters}
			/>
		{/if}
		{#if userRoleColumn}
			<DataTableFacetedFilter column={userRoleColumn} title={m.role()} options={userRoleFilters} />
		{/if}

		{#if isFiltered}
			<Button
				variant="ghost"
				onclick={() => {
					table.resetColumnFilters();
					table.resetGlobalFilter();
				}}
				class="h-8 px-2 lg:px-3"
			>
				{m.reset()}
				<XIcon />
			</Button>
		{/if}
	</div>

	<div class="flex items-center gap-2">
		{#if hasSelection && onRemoveSelected}
			<Button size="sm" onclick={() => onRemoveSelected?.(selectedIds!)}>
				{m.table_remove_selected({ count: selectedIds?.length ?? 0 })}
			</Button>
		{/if}
		<DataTableViewOptions {table} />
	</div>
</div>

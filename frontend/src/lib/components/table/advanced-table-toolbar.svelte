<script lang="ts" generics="TData extends Record<string, any>">
	import { Input } from '$lib/components/ui/input/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { ListRequestOptions } from '$lib/types/list-request.type';
	import { debounced } from '$lib/utils/debounce-util';
	import AdvancedTableColumnSelection from './advanced-table-column-selection.svelte';
	import AdvancedTableFilter from './advanced-table-filter.svelte';

	let {
		columns,
		visibleColumns = $bindable(),
		requestOptions,
		searchValue = $bindable(),
		withoutSearch = false,
		onFilterChange,
		refresh
	}: {
		columns: AdvancedTableColumn<TData>[];
		visibleColumns: string[];
		requestOptions: ListRequestOptions;
		searchValue?: string;
		withoutSearch?: boolean;
		onFilterChange?: (selected: Set<string | boolean>, column: string) => void;
		refresh: () => Promise<void>;
	} = $props();

	let filterableColumns = $derived(
		columns
			.filter((c) => c.filterableValues)
			.map((c) => ({
				name: c.label!,
				column: c.column!,
				options: c.filterableValues!
			}))
	);

	const onSearch = debounced(async (search: string) => {
		requestOptions.search = search;
		await refresh();
		searchValue = search;
	}, 300);
</script>

<div class="mb-4 flex flex-wrap items-end justify-between gap-2">
	<div class="flex flex-1 items-center gap-2 has-[>:nth-child(3)]:flex-wrap">
		{#if !withoutSearch}
			<Input
				value={searchValue}
				class="relative z-50 w-full sm:max-w-xs"
				placeholder={m.search()}
				type="text"
				oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
			/>
		{/if}

		{#each filterableColumns as col}
			<AdvancedTableFilter
				title={col.name}
				options={col.options}
				selectedValues={new Set(requestOptions.filters?.[col.column] || [])}
				onChanged={(selected) => onFilterChange?.(selected, col.column)}
			/>
		{/each}
		<AdvancedTableColumnSelection {columns} bind:selectedColumns={visibleColumns} />
	</div>
</div>

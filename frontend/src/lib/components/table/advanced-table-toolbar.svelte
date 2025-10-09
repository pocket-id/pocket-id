<script lang="ts" generics="TData extends Record<string, any>">
	import Button from '$lib/components/ui/button/button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { ListRequestOptions } from '$lib/types/list-request.type';
	import { debounced } from '$lib/utils/debounce-util';
	import XIcon from '@lucide/svelte/icons/x';
	import AdvancedTableColumnSelection from './advanced-table-column-selection.svelte';
	import AdvancedTableFilter from './advanced-table-filter.svelte';

	let {
		columns,
		visibleColumns = $bindable(),
		requestOptions = $bindable(),
		searchValue = $bindable(),
		withoutSearch = false,
		refresh
	}: {
		columns: AdvancedTableColumn<TData>[];
		visibleColumns: string[];
		requestOptions: ListRequestOptions;
		searchValue?: string;
		withoutSearch?: boolean;
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

	let isAnyFilterActive = $derived(
		Object.values(requestOptions.filters || {}).some((f) => f && f.length > 0)
	);

	const onSearch = debounced(async (search: string) => {
		requestOptions.search = search;
		await refresh();
		searchValue = search;
	}, 300);

	async function onFilterChange(selected: Set<string | boolean>, column: string) {
		requestOptions.filters = {
			...requestOptions.filters,
			[column]: Array.from(selected)
		};
		await refresh();
	}

	async function onFiltersReset() {
		requestOptions.filters = {};
		await refresh();
	}
</script>

<div class="mb-4 flex items-center justify-between">
	<div class="flex flex-1 items-center space-x-2">
		{#if !withoutSearch}
			<Input
				value={searchValue}
				class="relative z-50 max-w-sm"
				placeholder={m.search()}
				type="text"
				oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
			/>
		{/if}

		{#each filterableColumns as col}
			<AdvancedTableFilter
				title={col.name}
				options={col.options}
				selectedValues={new Set(requestOptions.filters![col.column] || [])}
				onChanged={(selected) => onFilterChange(selected, col.column)}
			/>
		{/each}

		{#if isAnyFilterActive}
			<Button variant="ghost" onclick={onFiltersReset} class="h-8 px-2 lg:px-3">
				{m.reset()}
				<XIcon />
			</Button>
		{/if}
	</div>

	<div class="flex items-center gap-2">
		<AdvancedTableColumnSelection {columns} bind:selectedColumns={visibleColumns} />
	</div>
</div>

<script lang="ts" generics="TData extends Record<string, any>">
	import * as InputGroup from '#lib/components/ui/input-group/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import { debounced } from '#lib/utils/debounce-util.ts';
	import { revealFocused } from '#lib/utils/event-util.ts';
	import { LucideSearch } from '@lucide/svelte';
	import AdvancedTableFilter from './advanced-table-filter.svelte';

	let {
		columns,
		requestOptions,
		searchValue,
		searchInput = $bindable(null),
		withoutSearch = false,
		onFilterChange,
		onSearch
	}: {
		columns: AdvancedTableColumn<TData>[];
		requestOptions: ListRequestOptions;
		searchValue: string;
		searchInput?: HTMLInputElement | null;
		withoutSearch?: boolean;
		onFilterChange: (column: string, value?: string | boolean) => void;
		onSearch: (search: string) => Promise<void>;
	} = $props();

	let filterableColumns = $derived(columns.filter((c) => c.filterableValues));

	const search = debounced((value: string) => onSearch(value), 300);
</script>

{#if !withoutSearch || filterableColumns.length > 0}
	<div class="mb-3 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:gap-3">
		{#if !withoutSearch}
			<InputGroup.Root class="w-full sm:w-75">
				<InputGroup.Input
					bind:ref={searchInput}
					value={searchValue}
					placeholder={m.search()}
					aria-label={m.search()}
					type="search"
					oninput={(e: Event) => search((e.currentTarget as HTMLInputElement).value)}
				/>
				<InputGroup.Addon><LucideSearch /></InputGroup.Addon>
			</InputGroup.Root>
		{/if}

		{#if filterableColumns.length > 0}
			<!-- The padding keeps focus rings visible inside the horizontal phone scroller -->
			<div
				role="group"
				aria-label={m.filters()}
				onfocusin={revealFocused}
				class="scroll-fade-x -m-1 flex scroll-px-1 items-center gap-1 p-1 max-sm:[scrollbar-width:none] max-sm:overflow-x-auto sm:flex-wrap"
			>
				{#each filterableColumns as col (col.column)}
					<AdvancedTableFilter
						title={col.label}
						options={col.filterableValues!}
						value={requestOptions.filters?.[col.column!]?.[0]}
						onChange={(value) => onFilterChange(col.column!, value)}
					/>
				{/each}
			</div>
		{/if}
	</div>
{/if}

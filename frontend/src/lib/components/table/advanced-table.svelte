<script lang="ts" generics="T extends {id:string}">
	import { goto } from '$app/navigation';
	import { page as currentPage } from '$app/state';
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import ListPagination from '#lib/components/list-pagination.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import Checkbox from '#lib/components/ui/checkbox/checkbox.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import Empty from '#lib/icons/empty.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions, Paginated, SortRequest } from '#lib/types/list-request.type.ts';
	import { cn } from '#lib/utils/style.ts';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import { PersistedState } from 'runed';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Skeleton } from '../ui/skeleton';
	import AdvancedTableColumnSelection from './advanced-table-column-selection.svelte';
	import AdvancedTableToolbar from './advanced-table-toolbar.svelte';

	let {
		id,
		selectedIds = $bindable(),
		withoutSearch = false,
		query: externalQuery,
		filtered = false,
		onClearQuery,
		onRowClick,
		rowSelectionDisabled,
		fetchCallback,
		defaultSort,
		columns,
		actions
	}: {
		id: string;
		selectedIds?: string[];
		withoutSearch?: boolean;
		// The search of a page that brings its own search field, so no matches can be told apart from no data
		query?: string;
		// Whether a filter the page applies itself, outside the table's own facets, narrows the list
		filtered?: boolean;
		onClearQuery?: () => void;
		fetchCallback: (requestOptions: ListRequestOptions) => Promise<Paginated<T>>;
		defaultSort?: SortRequest;
		columns: AdvancedTableColumn<T>[];
		onRowClick?: (item: T) => void;
		actions?: CreateAdvancedTableActions<T>;
	} = $props();

	let items: Paginated<T> | undefined = $state();
	let searchValue = $state('');
	let isRefreshing = $state(false);
	let searchInput = $state<HTMLInputElement | null>(null);
	const query = $derived((externalQuery ?? searchValue).trim());
	const hasOptionalColumns = columns.some((c) => c.hidden);
	const hasStickyColumn = !!actions || hasOptionalColumns;

	const availablePageSizes: number[] = [20, 50, 100];

	type TablePreferences = {
		visibleColumns: string[];
		paginationLimit: number;
		sort?: SortRequest;
		filters?: Record<string, (string | boolean)[]>;
		length?: number;
	};

	const tablePreferences = new PersistedState<TablePreferences>(`table-${id}-preferences`, {
		visibleColumns: columns.filter((c) => !c.hidden).map((c) => c.column ?? c.key!),
		paginationLimit: 20,
		filters: initializeFilters()
	});

	const requestOptions = $state<ListRequestOptions>({
		sort: tablePreferences.current.sort ?? defaultSort,
		pagination: { limit: tablePreferences.current.paginationLimit, page: 1 },
		filters: tablePreferences.current.filters
	});

	let visibleColumns = $derived(
		columns.filter(
			(c) => tablePreferences.current.visibleColumns?.includes(c.column ?? c.key!) ?? []
		)
	);

	onMount(async () => {
		const urlParams = new URLSearchParams(window.location.search);
		const page = parseInt(urlParams.get(`${id}-page`) ?? '') || undefined;
		if (page) {
			requestOptions.pagination!.page = page;
		}
		await refresh();
	});

	let allChecked = $derived.by(() => {
		if (!selectedIds || !items || items.data.length === 0) return false;
		for (const item of items!.data) {
			if (!selectedIds.includes(item.id)) {
				return false;
			}
		}
		return true;
	});

	async function onAllCheck(checked: boolean) {
		const pageIds = items!.data
			.filter((item) => !rowSelectionDisabled?.(item))
			.map((item) => item.id);
		const current = selectedIds ?? [];

		if (checked) {
			selectedIds = Array.from(new Set([...current, ...pageIds]));
		} else {
			selectedIds = current.filter((id) => !pageIds.includes(id));
		}
	}

	async function onCheck(checked: boolean, id: string) {
		const current = selectedIds ?? [];
		if (checked) {
			selectedIds = Array.from(new Set([...current, id]));
		} else {
			selectedIds = current.filter((selectedId) => selectedId !== id);
		}
	}

	async function onPageChange(page: number) {
		await changePageState(page);
		await refresh();
	}

	async function onPageSizeChange(size: number) {
		requestOptions.pagination = { limit: size, page: 1 };
		tablePreferences.current.paginationLimit = size;
		await refresh();
	}

	async function onFilterChange(column: string, value?: string | boolean) {
		requestOptions.filters = {
			...requestOptions.filters,
			[column]: value === undefined ? [] : [value]
		};
		tablePreferences.current.filters = requestOptions.filters;
		await refresh();
	}

	async function onSort(column?: string) {
		if (!column) return;

		const isSameColumn = requestOptions.sort?.column === column;
		const nextDirection: 'asc' | 'desc' =
			isSameColumn && requestOptions.sort?.direction === 'asc' ? 'desc' : 'asc';

		requestOptions.sort = { column, direction: nextDirection };
		tablePreferences.current.sort = requestOptions.sort;
		await refresh();
	}

	async function changePageState(page: number) {
		const url = new URL(window.location.href);
		url.searchParams.set(`${id}-page`, page.toString());
		await goto(url, { shallow: true, replace: true, state: currentPage.state });
		requestOptions.pagination!.page = page;
	}

	function updateListLength(totalItems: number) {
		// Searches and a page's own filters are not kept between visits, so their results must not decide what the next visit shows while it loads
		if (query || filtered) return;
		tablePreferences.current.length =
			totalItems > tablePreferences.current.paginationLimit
				? tablePreferences.current.paginationLimit
				: totalItems;
	}

	function initializeFilters() {
		const filters: Record<string, (string | boolean)[]> = {};
		columns.forEach((c) => {
			if (c.filterableValues) {
				filters[c.column!] = [];
			}
		});
		return filters;
	}

	// Phones show only the first column, which takes the free width and truncates inside it, so the table never scrolls sideways
	// Wider screens give it the width its content is capped at, so the columns stay put when paging, sorting or searching
	// Narrower tables drop the columns that give way first, so the rest still fit beside the names
	const columnClass = (column: AdvancedTableColumn<T>, index: number) =>
		cn(
			index === 0
				? 'max-sm:w-full max-sm:max-w-0 sm:w-[calc(var(--table-id-max)+2rem)]'
				: 'max-sm:hidden',
			index === 0 && !hasStickyColumn && 'max-sm:rounded-r-sm',
			column.hideBelow === 900 && '@max-[900px]:hidden',
			column.hideBelow === 760 && '@max-[760px]:hidden',
			column.hideBelow === 520 && '@max-[520px]:hidden'
		);

	function getPrimaryAction(item: T) {
		if (!actions) return null;
		const availableActions = actions(item).filter((a) => a.primary);
		return availableActions.length > 0 ? () => availableActions[0].onClick(item) : null;
	}

	// One action clears the search and the facets together, and focus moves to the search because the action removes itself
	async function clearQuery() {
		if (onClearQuery) return onClearQuery();
		searchValue = '';
		requestOptions.search = '';
		requestOptions.filters = initializeFilters();
		tablePreferences.current.filters = requestOptions.filters;
		searchInput?.focus();
		await refresh();
	}

	export async function refresh() {
		isRefreshing = true;
		try {
			items = await fetchCallback(requestOptions);
		} finally {
			isRefreshing = false;
		}
		await changePageState(items.pagination.currentPage);
		updateListLength(items.pagination.totalItems);
	}
</script>

<AdvancedTableToolbar
	{columns}
	{requestOptions}
	{searchValue}
	bind:searchInput
	{withoutSearch}
	{refresh}
	{onFilterChange}
/>

{#if !items}
	<!-- Fast responses replace the skeleton before it fades in, so it never flashes -->
	<div aria-busy="true" class="transition-opacity delay-(--t-skeleton-delay) starting:opacity-0">
		<Skeleton class="h-10 w-full rounded-sm" />
		{#each Array.from({ length: tablePreferences.current.length || 5 }, (_, i) => i) as i (i)}
			<div class="flex h-(--row-2line) items-center border-b px-4">
				<Skeleton class="h-3.5 w-2/5" />
			</div>
		{/each}
	</div>
{:else}
	{#if !items}
		<div aria-busy="true">
			<Skeleton class="h-10 w-full rounded-sm" />
			{#each Array.from({ length: tablePreferences.current.length || 10 }, (_, i) => i) as i (i)}
				<div class="flex h-(--row-2line) items-center border-b px-4">
					<Skeleton class="h-3.5 w-2/5" />
				</div>
			{/each}
		</div>
	{:else}
		<!-- Refetches keep the old rows dimmed until the new page arrives, after a short delay so fast responses never flash -->
		<div
			in:fade={{ duration: 120 }}
			aria-busy={isRefreshing}
			class={cn(
				'@container transition-opacity duration-(--dur-base)',
				isRefreshing && 'pointer-events-none opacity-50 delay-(--t-dim-delay)'
			)}
		>
			<Table.Root class="min-w-full table-auto">
				<Table.Header>
					<Table.Row>
						{#if selectedIds}
							<!-- Selecting a row must not also open it -->
							<Table.Cell class="w-10" onclick={(e) => e.stopPropagation()}>
								<Checkbox
									checked={selected}
									aria-labelledby={nameId}
									onCheckedChange={(c: boolean) => onCheck(c, item.id)}
								/>
							</Table.Head>
						{/if}
						{#each visibleColumns as column, i (column.key ?? column.column ?? column.label)}
							<Table.Cell class={columnClass(column, i)} id={i === 0 ? nameId : undefined}>
								{#if column.value}
									{column.value(item)}
								{:else if column.cell}
									{@render column.cell({ item })}
								{:else if column.column && typeof item[column.column] === 'boolean'}
									{item[column.column] ? m.yes() : m.no()}
								{:else if column.column}
									{item[column.column]}
								{/if}
							</Table.Head>
						{/each}
						<!-- Pinned to the right edge so the actions stay reachable when the table scrolls horizontally -->
						<Table.Head class="sticky right-0 z-(--z-sticky) w-px pr-2 pl-2 text-right">
							<AdvancedTableColumnSelection
								{columns}
								bind:selectedColumns={tablePreferences.current.visibleColumns}
							/>
						</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each items.data as item (item.id)}
						{@const selected = selectedIds?.includes(item.id)}
						<Table.Row
							data-state={selected ? 'selected' : undefined}
							class={cn('group/row', (getPrimaryAction(item) || onRowClick) && 'cursor-pointer')}
							onclick={onRowClick ? () => onRowClick(item) : getPrimaryAction(item)}
						>
							{#if selectedIds}
								<Table.Cell class="w-10">
									<Checkbox
										disabled={selectionDisabled || rowSelectionDisabled?.(item)}
										checked={selected}
										onCheckedChange={(c: boolean) => onCheck(c, item.id)}
									/>
								</Table.Cell>
							{/if}
							{#each visibleColumns as column, i (column.key ?? column.column ?? column.label)}
								<Table.Cell class={columnClass(column, i)}>
									{#if column.value}
										{column.value(item)}
									{:else if column.cell}
										{@render column.cell({ item })}
									{:else if column.column && typeof item[column.column] === 'boolean'}
										{item[column.column] ? m.yes() : m.no()}
									{:else if column.column}
										{item[column.column]}
									{/if}
								</Table.Cell>
							{/each}
							<!-- The sticky cell is opaque so scrolled cells pass under it, so it repeats the row's hover and selected fills -->
							<Table.Cell
								class="bg-surface group-hover/row:bg-fill-row-hover group-has-focus-visible/row:bg-fill-row-hover group-data-[state=selected]/row:bg-fill-muted sticky right-0 z-(--z-sticky) w-px py-0 pr-2 pl-2 text-right transition-colors duration-(--dur-fast)"
							>
								{#if actions}
									<ActionsMenu
										{item}
										actions={actions(item)}
										label={m.actions_for_name({ name: String(item[columns[0].column!]) })}
										size="icon-sm"
									/>
								{/if}
							</Table.Cell>
						{/if}
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		{#if items.data.length === 0}
			<p
				class="text-control text-fg-muted flex min-h-(--row-2line) flex-wrap items-center gap-x-3 border-b px-4 wrap-anywhere"
			>
				{#if query}
					{m.no_matches_for_query({ query })}
				{:else if narrowed}
					{m.no_matches_for_filters()}
				{:else}
					{m.no_items_found()}
				{/if}
				{#if narrowed}
					<Button variant="link" onclick={clearQuery}>
						{query ? m.clear_search() : m.clear_filters()}
					</Button>
				{/if}
			</p>
		{/if}
	</div>
{/if}

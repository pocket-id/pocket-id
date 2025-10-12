<script lang="ts" generics="T extends {id:string}">
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import * as Pagination from '$lib/components/ui/pagination';
	import * as Select from '$lib/components/ui/select';
	import * as Table from '$lib/components/ui/table/index.js';
	import Empty from '$lib/icons/empty.svelte';
	import { m } from '$lib/paraglide/messages';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { ListRequestOptions, Paginated, SortRequest } from '$lib/types/list-request.type';
	import { cn } from '$lib/utils/style';
	import { ChevronDown, LucideEllipsis } from '@lucide/svelte';
	import { PersistedState } from 'runed';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import Button, { buttonVariants } from '../ui/button/button.svelte';
	import * as DropdownMenu from '../ui/dropdown-menu/index.js';
	import { Skeleton } from '../ui/skeleton';
	import AdvancedTableToolbar from './advanced-table-toolbar.svelte';

	let {
		id,
		selectedIds = $bindable(),
		withoutSearch = false,
		selectionDisabled = false,
		fetchCallback,
		defaultSort,
		columns,
		actions
	}: {
		id: string;
		selectedIds?: string[];
		withoutSearch?: boolean;
		selectionDisabled?: boolean;
		fetchCallback: (requestOptions: ListRequestOptions) => Promise<Paginated<T>>;
		defaultSort?: SortRequest;
		columns: AdvancedTableColumn<T>[];
		actions?: CreateAdvancedTableActions<T>;
	} = $props();

	let items: Paginated<T> | undefined = $state();
	let searchValue = $state('');

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
		const pageIds = items!.data.map((item) => item.id);
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
		changePageState(page);
		await refresh();
	}

	async function onPageSizeChange(size: number) {
		requestOptions.pagination = { limit: size, page: 1 };
		tablePreferences.current.paginationLimit = size;
		await refresh();
	}

	async function onFilterChange(selected: Set<string | boolean>, column: string) {
		requestOptions.filters = {
			...requestOptions.filters,
			[column]: Array.from(selected)
		};
		tablePreferences.current.filters = requestOptions.filters;
		await refresh();
	}

	async function onSort(column?: string, direction: 'asc' | 'desc' = 'asc') {
		if (!column) return;

		requestOptions.sort = { column, direction };
		tablePreferences.current.sort = requestOptions.sort;
		await refresh();
	}

	function changePageState(page: number) {
		const url = new URL(window.location.href);
		url.searchParams.set(`${id}-page`, page.toString());
		history.replaceState(history.state, '', url.toString());
		requestOptions.pagination!.page = page;
	}

	function updateListLength(totalItems: number) {
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

	export async function refresh() {
		items = await fetchCallback(requestOptions);
		changePageState(items.pagination.currentPage);
		updateListLength(items.pagination.totalItems);
	}
</script>

<AdvancedTableToolbar
	{columns}
	bind:visibleColumns={tablePreferences.current.visibleColumns}
	{requestOptions}
	{searchValue}
	{withoutSearch}
	{refresh}
	{onFilterChange}
/>

{#if (items?.pagination.totalItems === 0 && searchValue === '') || tablePreferences.current.length === 0}
	<div class="my-5 flex flex-col items-center">
		<Empty class="text-muted-foreground h-20" />
		<p class="text-muted-foreground mt-3 text-sm">{m.no_items_found()}</p>
	</div>
{:else}
	{#if !items}
		<div>
			{#each Array((tablePreferences.current.length || 10) + 1) as _}
				<div>
					<Skeleton class="mt-3 h-[45px] w-full rounded-lg" />
				</div>
			{/each}
		</div>
	{:else}
		<div in:fade>
			<Table.Root class="min-w-full table-auto overflow-x-auto">
				<Table.Header>
					<Table.Row>
						{#if selectedIds}
							<Table.Head class="w-12">
								<Checkbox
									disabled={selectionDisabled}
									checked={allChecked}
									onCheckedChange={(c: boolean) => onAllCheck(c as boolean)}
								/>
							</Table.Head>
						{/if}

						{#each visibleColumns as column}
							<Table.Head class={cn(column.sortable && 'p-0')}>
								{#if column.sortable}
									<Button
										variant="ghost"
										class="h-12 w-full justify-start px-4 font-medium hover:bg-transparent"
										onclick={() =>
											onSort(
												column.column,
												requestOptions.sort?.direction === 'desc' ? 'asc' : 'desc'
											)}
									>
										<span class="flex items-center">
											{column.label}
											<ChevronDown
												class={cn(
													'ml-2 size-4 transition-all',
													requestOptions.sort?.column === column.column
														? requestOptions.sort?.direction === 'asc'
															? 'rotate-180 opacity-100'
															: 'opacity-100'
														: 'opacity-0'
												)}
											/>
										</span>
									</Button>
								{:else}
									{column.label}
								{/if}
							</Table.Head>
						{/each}
						{#if actions}
							<Table.Head align="right" class="w-12">
								<span class="sr-only">{m.actions()}</span>
							</Table.Head>
						{/if}
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each items.data as item}
						<Table.Row class={selectedIds?.includes(item.id) ? 'bg-muted/20' : ''}>
							{#if selectedIds}
								<Table.Cell class="w-12">
									<Checkbox
										disabled={selectionDisabled}
										checked={selectedIds.includes(item.id)}
										onCheckedChange={(c: boolean) => onCheck(c, item.id)}
									/>
								</Table.Cell>
							{/if}
							{#each visibleColumns as column}
								<Table.Cell>
									{#if column.value}
										{column.value(item)}
									{:else if column.cell}
										{@render column.cell({ item })}
									{:else if column.column && typeof item[column.column] === 'boolean'}
										{item[column.column] ? m.enabled() : m.disabled()}
									{:else if column.column}
										{item[column.column]}
									{/if}
								</Table.Cell>
							{/each}
							{#if actions}
								<Table.Cell align="right" class="w-12 py-0">
									<DropdownMenu.Root>
										<DropdownMenu.Trigger
											class={buttonVariants({ variant: 'ghost', size: 'icon' })}
										>
											<LucideEllipsis class="size-4" />
											<span class="sr-only">{m.toggle_menu()}</span>
										</DropdownMenu.Trigger>
										<DropdownMenu.Content align="end">
											{#each actions(item).filter((a) => !a.visible || a.visible) as action}
												<DropdownMenu.Item
													onclick={() => action.onClick(item)}
													disabled={action.disabled}
													class={action.variant === 'danger'
														? 'text-red-500 focus:!text-red-700'
														: ''}
												>
													{#if action.icon}
														{@const Icon = action.icon}
														<Icon class="mr-2 size-4" />
													{/if}
													{action.label}
												</DropdownMenu.Item>
											{/each}
										</DropdownMenu.Content>
									</DropdownMenu.Root>
								</Table.Cell>
							{/if}
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</div>
	{/if}

	<div class="mt-5 flex flex-col-reverse items-center justify-between gap-3 sm:flex-row">
		<div class="flex items-center space-x-2">
			<p class="text-sm font-medium">{m.items_per_page()}</p>
			<Select.Root
				type="single"
				value={items?.pagination.itemsPerPage.toString()}
				onValueChange={(v) => onPageSizeChange(Number(v))}
			>
				<Select.Trigger class="h-9 w-[80px]">
					{items?.pagination.itemsPerPage}
				</Select.Trigger>
				<Select.Content>
					{#each availablePageSizes as size}
						<Select.Item value={size.toString()}>{size}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Pagination.Root
			class="mx-0 w-auto"
			count={items?.pagination.totalItems || 0}
			perPage={items?.pagination.itemsPerPage}
			{onPageChange}
			page={items?.pagination.currentPage}
		>
			{#snippet children({ pages })}
				<Pagination.Content class="flex justify-end">
					<Pagination.Item>
						<Pagination.PrevButton />
					</Pagination.Item>
					{#each pages as page (page.key)}
						{#if page.type !== 'ellipsis' && page.value != 0}
							<Pagination.Item>
								<Pagination.Link {page} isActive={items?.pagination.currentPage === page.value}>
									{page.value}
								</Pagination.Link>
							</Pagination.Item>
						{/if}
					{/each}
					<Pagination.Item>
						<Pagination.NextButton />
					</Pagination.Item>
				</Pagination.Content>
			{/snippet}
		</Pagination.Root>
	</div>
{/if}

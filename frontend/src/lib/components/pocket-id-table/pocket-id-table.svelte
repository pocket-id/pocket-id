<script lang="ts" generics="TData extends {id: string}">
	import {
		type Column,
		type ColumnDef,
		type ColumnFiltersState,
		type Row,
		type RowSelectionState,
		type SortingState,
		type VisibilityState,
		type Table as TableType,
		getCoreRowModel,
		getFacetedRowModel,
		getFacetedUniqueValues,
		getFilteredRowModel
	} from '@tanstack/table-core';
	import DataTableToolbar from './pocket-id-table-toolbar.svelte';
	import { createSvelteTable } from '$lib/components/ui/data-table/data-table.svelte.js';
	import FlexRender from '$lib/components/ui/data-table/flex-render.svelte';
	import * as Table from '$lib/components/ui/table/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { renderComponent, renderSnippet } from '$lib/components/ui/data-table/render-helpers.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronsLeftIcon from '@lucide/svelte/icons/chevrons-left';
	import ChevronsRightIcon from '@lucide/svelte/icons/chevrons-right';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import * as Select from '$lib/components/ui/select/index.js';
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/style';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { Snippet } from 'svelte';
	import type { ColumnSpec } from './pocket-id-table.types.svelte';
	import TableCheckbox from './pocket-id-table-checkbox.svelte';
	import { m } from '$lib/paraglide/messages';
	import { PersistedState } from 'runed';
	import type { FilterMap } from '$lib/types/pagination.type';
	import {
		type CompactTablePrefs,
		encodeHidden,
		applyHiddenPatch,
		encodeFilters,
		decodeFilters
	} from './pocket-id-table.types.svelte';

	let {
		items,
		requestOptions = $bindable(),
		withoutSearch = $bindable(),
		withoutPagination = false,
		selectionDisabled = false,
		onRefresh,
		columns,
		rowActions,
		selectedIds = $bindable<string[]>([]),
		onRemoveSelected,
		persistKey
	}: {
		items: Paginated<TData>;
		requestOptions: SearchPaginationSortRequest;
		withoutSearch?: boolean;
		withoutPagination?: boolean;
		selectionDisabled?: boolean;
		onRefresh: (requestOptions: SearchPaginationSortRequest) => Promise<Paginated<TData>>;
		columns: ColumnSpec<TData>[];
		rowActions?: Snippet<[{ row: Row<TData>; item: TData }]>;
		selectedIds?: string[];
		onRemoveSelected?: (ids: string[]) => void;
		persistKey?: string;
	} = $props();

	let rowSelection = $state<RowSelectionState>({});
	let columnVisibility = $state<VisibilityState>({});
	let columnFilters = $state<ColumnFiltersState>([]);
	let sorting = $state<SortingState>([]);
	let globalFilter = $state<string>('');

	const enablePersist = !!persistKey;

	const getDefaultLimit = () =>
		requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 20;
	const prefs = enablePersist
		? new PersistedState<CompactTablePrefs>(
				persistKey as string,
				{ v: [], f: [], g: '', l: getDefaultLimit() },
				{ syncTabs: false }
			)
		: null;

	const passAllGlobal: (row: unknown, columnId: string, filterValue: unknown) => boolean = () =>
		true;

	const currentPage = $derived(
		items.pagination.currentPage ?? requestOptions?.pagination?.page ?? 1
	);
	const totalPages = $derived(items.pagination.totalPages ?? 1);
	const totalItems = $derived(items.pagination.totalItems ?? 0);
	const pageSize = $derived(
		requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 20
	);
	const canPrev = $derived(currentPage > 1);
	const canNext = $derived(currentPage < totalPages);

	// Apply persisted state on mount
	import { onMount } from 'svelte';
	onMount(() => {
		if (!enablePersist) return;
		const cur = prefs?.current ?? { v: [], f: [], g: '', l: getDefaultLimit() };

		applyHiddenPatch(columnVisibility, cur.v);

		// Filters
		let shouldRefresh = false;
		const restoredFilters = decodeFilters(cur.f);
		if (restoredFilters.length) {
			columnFilters = restoredFilters;
			// Send restored filters to backend
			requestOptions = {
				...requestOptions,
				filters: toFilterMap(restoredFilters),
				pagination: { page: 1, limit: requestOptions?.pagination?.limit ?? getDefaultLimit() }
			};
			shouldRefresh = true;
		}
		if (cur.g && cur.g !== globalFilter) {
			globalFilter = cur.g;
			requestOptions = {
				...requestOptions,
				search: cur.g,
				pagination: { page: 1, limit: requestOptions?.pagination?.limit ?? getDefaultLimit() }
			};
			shouldRefresh = true;
		}
		// Page size
		const persistedLimit = cur.l ?? getDefaultLimit();
		const currentLimit = requestOptions?.pagination?.limit ?? getDefaultLimit();
		if (persistedLimit !== currentLimit) {
			requestOptions = { ...requestOptions, pagination: { page: 1, limit: persistedLimit } };
			shouldRefresh = true;
		}

		const urlParams = new URLSearchParams(window.location.search);
		const page = parseInt(urlParams.get(`${persistKey}-page`) ?? '') || undefined;
		if (page) {
			requestOptions.pagination!.page = page;
			shouldRefresh = true;
		}

		if (shouldRefresh) onRefresh(requestOptions);
	});

	function updatePagination(patch: Partial<{ page: number; limit: number }>) {
		const prev = requestOptions?.pagination ?? {
			page: items?.pagination?.currentPage ?? 1,
			limit: items?.pagination?.itemsPerPage ?? 10
		};
		const next = { ...prev, ...patch };
		requestOptions = { ...requestOptions, pagination: next };
		onRefresh(requestOptions);
		changePageUrlState(next.page);
	}

	function setPage(page: number) {
		if (page < 1) page = 1;
		if (totalPages > 0 && page > totalPages) page = totalPages;
		updatePagination({ page });
	}

	function changePageUrlState(page: number) {
		const url = new URL(window.location.href);
		url.searchParams.set(`${persistKey}-page`, page.toString());
		history.replaceState(history.state, '', url.toString());
	}

	function setPageSize(limit: number) {
		// Persist page size
		if (enablePersist && prefs) prefs.current = { ...prefs.current, l: limit };
		updatePagination({ limit, page: 1 });
	}

	function onToggleAll(checked: boolean, table: TableType<TData>) {
		const pageIds = table.getRowModel().rows.map((r) => (r.original as TData).id);
		if (checked) {
			const set = new Set([...(selectedIds ?? []), ...pageIds]);
			selectedIds = Array.from(set);
		} else {
			const pageSet = new Set(pageIds);
			selectedIds = (selectedIds ?? []).filter((id) => !pageSet.has(id));
		}
	}

	function toFilterMap(filters: ColumnFiltersState): FilterMap {
		const out: FilterMap = {};
		for (const f of filters ?? []) {
			const id = f.id;
			let v: unknown = (f as any).value;

			// If the UI stores arrays/Sets, take the first selected value only
			if (Array.isArray(v)) {
				if (v.length === 0) continue;
				v = v[0];
			} else if (v && typeof v === 'object' && v instanceof Set) {
				const first = (v as Set<unknown>).values().next().value;
				if (first === undefined) continue;
				v = first;
			}

			if (v !== undefined && v !== null && String(v).trim() !== '') {
				out[id] = v as any; // scalar only
			}
		}
		return out;
	}

	function onToggleRow(checked: boolean, id: string) {
		if (checked) {
			if (!selectedIds?.includes(id)) selectedIds = [...(selectedIds ?? []), id];
		} else {
			selectedIds = (selectedIds ?? []).filter((x) => x !== id);
		}
	}

	function buildColumns(specs: ColumnSpec<TData>[]): ColumnDef<TData>[] {
		const cols: ColumnDef<TData>[] = [];

		if (!selectionDisabled) {
			cols.push({
				id: 'select',
				header: ({ table }) => {
					const pageIds = table.getRowModel().rows.map((r) => (r.original as TData).id);
					const selectedSet = new Set(selectedIds ?? []);
					const total = pageIds.length;
					const selectedOnPage = pageIds.filter((id) => selectedSet.has(id)).length;
					const checked = total > 0 && selectedOnPage === total;
					const indeterminate = selectedOnPage > 0 && selectedOnPage < total;

					return renderComponent(TableCheckbox, {
						checked,
						indeterminate,
						onCheckedChange: (value) => onToggleAll(!!value, table),
						'aria-label': m.select_all()
					});
				},
				cell: ({ row }) => {
					const id = (row.original as TData).id;
					return renderComponent(TableCheckbox, {
						checked: (selectedIds ?? []).includes(id),
						onCheckedChange: (value) => onToggleRow(!!value, id),
						'aria-label': m.select_row()
					});
				},
				enableSorting: false,
				enableHiding: false
			});
		}

		specs.forEach((spec, i) => {
			const accessorKey = spec.accessorKey;
			const accessorFn = spec.accessorFn;
			const id = spec.id ?? (accessorKey as string) ?? `col_${i}`;

			cols.push({
				id,
				...(accessorKey ? { accessorKey } : {}),
				...(accessorFn ? { accessorFn } : {}),
				header: ({ column }) => {
					if (spec.header)
						return renderSnippet(spec.header, { column, title: spec.title, class: spec.class });
					if (spec.sortable)
						return renderSnippet(ColumnHeader, { column, title: spec.title, class: spec.class });
					return renderSnippet(PlainHeader, { title: spec.title, class: spec.class });
				},
				cell: ({ row, getValue }) => {
					const item = row.original as TData;
					const value = accessorKey ? row.getValue(accessorKey) : getValue?.();
					if (spec.cell) return renderSnippet(spec.cell, { row, item, value });
					return renderSnippet(TextCell, { value });
				},
				enableSorting: !!spec.sortable,
				enableHiding: true,
				filterFn: spec.filterFn
			});

			if (spec.hidden) {
				columnVisibility[String(accessorKey ?? id)] = false;
			}
		});

		if (rowActions) {
			cols.push({
				id: 'actions',
				cell: ({ row }) => renderSnippet(rowActions, { row, item: row.original as TData })
			});
		}

		return cols;
	}

	const columnsDef: ColumnDef<TData>[] = buildColumns(columns ?? []);

	const table = createSvelteTable({
		get data() {
			return items.data ?? [];
		},
		state: {
			get sorting() {
				return sorting;
			},
			get columnVisibility() {
				return columnVisibility;
			},
			get rowSelection() {
				return rowSelection;
			},
			get columnFilters() {
				return columnFilters;
			},
			get globalFilter() {
				return globalFilter;
			}
		},
		columns: columnsDef,
		globalFilterFn: passAllGlobal,
		enableRowSelection: !selectionDisabled,
		onRowSelectionChange: (updater) => {
			rowSelection = typeof updater === 'function' ? updater(rowSelection) : updater;
		},
		onSortingChange: (updater) => {
			const next = typeof updater === 'function' ? updater(sorting) : updater;
			sorting = next;
			const first = next[0];
			if (first) {
				requestOptions = {
					...requestOptions,
					sort: { column: String(first.id), direction: first.desc ? 'desc' : 'asc' },
					pagination: {
						page: 1,
						limit: requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 10
					}
				};
			} else {
				requestOptions = {
					...requestOptions,
					sort: undefined,
					pagination: {
						page: 1,
						limit: requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 10
					}
				};
			}
			onRefresh(requestOptions);
		},
		onColumnFiltersChange: (updater) => {
			columnFilters = typeof updater === 'function' ? updater(columnFilters) : updater;

			// Persist column filters
			if (enablePersist && prefs) {
				prefs.current = { ...prefs.current, f: encodeFilters(columnFilters) };
			}

			// Send to backend and refresh
			requestOptions = {
				...requestOptions,
				filters: toFilterMap(columnFilters),
				pagination: {
					page: 1,
					limit: requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 10
				}
			};
			onRefresh(requestOptions);
		},
		onColumnVisibilityChange: (updater) => {
			columnVisibility = typeof updater === 'function' ? updater(columnVisibility) : updater;
			// Persist visibility
			if (enablePersist && prefs) {
				prefs.current = { ...prefs.current, v: encodeHidden(columnVisibility) };
			}
		},
		onGlobalFilterChange: (value) => {
			globalFilter = (value ?? '') as string;
			const limit = requestOptions?.pagination?.limit ?? items?.pagination?.itemsPerPage ?? 10;
			requestOptions = {
				...requestOptions,
				search: globalFilter,
				pagination: { page: 1, limit }
			};
			// Persist global filter
			if (enablePersist && prefs) {
				prefs.current = { ...prefs.current, g: globalFilter };
			}
			onRefresh(requestOptions);
		},
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getFacetedRowModel: getFacetedRowModel(),
		getFacetedUniqueValues: getFacetedUniqueValues()
	});

	$effect(() => {
		const s = requestOptions?.sort;
		if (!s) {
			if (sorting.length) sorting = [];
			return;
		}
		const desc = s.direction === 'desc';
		if (!sorting[0] || sorting[0].id !== s.column || sorting[0].desc !== desc) {
			sorting = [{ id: s.column, desc }];
		}
	});
</script>

{#snippet TextCell({ value }: { value: unknown })}
	<span class="max-w-[500px] truncate">{value ?? ''}</span>
{/snippet}

{#snippet PlainHeader({
	title,
	class: className,
	...restProps
}: { title: string } & HTMLAttributes<HTMLDivElement>)}
	<div class={className} {...restProps}>{title}</div>
{/snippet}

{#snippet Pagination({ table }: { table: TableType<TData> })}
	<div class="flex items-center justify-between px-2">
		<div class="text-muted-foreground flex-1 text-sm">
			{m.table_showing_of({ visible: table.getFilteredRowModel().rows.length, total: totalItems })}
		</div>
		<div class="flex items-center space-x-6 lg:space-x-8">
			<div class="flex items-center space-x-2">
				<p class="text-sm font-medium">{m.items_per_page()}</p>
				<Select.Root
					allowDeselect={false}
					type="single"
					value={`${pageSize}`}
					onValueChange={(value) => setPageSize(Number(value))}
				>
					<Select.Trigger class="h-8 w-[70px]">
						{String(pageSize)}
					</Select.Trigger>
					<Select.Content side="top">
						{#each [10, 20, 30, 40, 50] as size (size)}
							<Select.Item value={`${size}`}>
								{size}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div class="flex w-[100px] items-center justify-center text-sm font-medium">
				{m.table_page_of({ page: currentPage, totalPages })}
			</div>
			<div class="flex items-center space-x-2">
				<Button
					variant="outline"
					class="hidden size-8 p-0 lg:flex"
					onclick={() => setPage(1)}
					disabled={!canPrev}
				>
					<ChevronsLeftIcon />
				</Button>
				<Button
					variant="outline"
					class="size-8 p-0"
					onclick={() => setPage(currentPage - 1)}
					disabled={!canPrev}
				>
					<ChevronLeftIcon />
				</Button>
				<Button
					variant="outline"
					class="size-8 p-0"
					onclick={() => setPage(currentPage + 1)}
					disabled={!canNext}
				>
					<ChevronRightIcon />
				</Button>
				<Button
					variant="outline"
					class="hidden size-8 p-0 lg:flex"
					onclick={() => setPage(totalPages)}
					disabled={!canNext}
				>
					<ChevronsRightIcon />
				</Button>
			</div>
		</div>
	</div>
{/snippet}

{#snippet ColumnHeader({
	column,
	title,
	class: className,
	...restProps
}: { column: Column<TData>; title: string } & HTMLAttributes<HTMLDivElement>)}
	{#if !column?.getCanSort()}
		<div class={className} {...restProps}>
			{title}
		</div>
	{:else}
		<div class={cn('flex items-center', className)} {...restProps}>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant="ghost"
							size="sm"
							class="data-[state=open]:bg-accent -ml-3 h-8"
						>
							<span>
								{title}
							</span>
							{#if column.getIsSorted() === 'desc'}
								<ArrowDownIcon />
							{:else if column.getIsSorted() === 'asc'}
								<ArrowUpIcon />
							{:else}
								<ChevronsUpDownIcon />
							{/if}
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="start">
					<DropdownMenu.Item onclick={() => column.toggleSorting(false)}>
						<ArrowUpIcon class="text-muted-foreground/70 mr-2 size-3.5" />
						{m.asc()}
					</DropdownMenu.Item>
					<DropdownMenu.Item onclick={() => column.toggleSorting(true)}>
						<ArrowDownIcon class="text-muted-foreground/70 mr-2 size-3.5" />
						{m.desc()}
					</DropdownMenu.Item>
					<DropdownMenu.Separator />
					<DropdownMenu.Item onclick={() => column.toggleVisibility(false)}>
						<EyeOffIcon class="text-muted-foreground/70 mr-2 size-3.5" />
						{m.hide()}
					</DropdownMenu.Item>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>
	{/if}
{/snippet}

<div class="space-y-4">
	{#if !withoutSearch}
		<DataTableToolbar {table} {selectedIds} {selectionDisabled} {onRemoveSelected} />
	{/if}
	<div class="rounded-md">
		<Table.Root>
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row>
						{#each headerGroup.headers as header (header.id)}
							<Table.Head colspan={header.colSpan}>
								{#if !header.isPlaceholder}
									<FlexRender
										content={header.column.columnDef.header}
										context={header.getContext()}
									/>
								{/if}
							</Table.Head>
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#each table.getRowModel().rows as row (row.id)}
					<Table.Row
						data-state={(selectedIds ?? []).includes((row.original as TData).id) && 'selected'}
					>
						{#each row.getVisibleCells() as cell (cell.id)}
							<Table.Cell>
								<FlexRender content={cell.column.columnDef.cell} context={cell.getContext()} />
							</Table.Cell>
						{/each}
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={columnsDef.length} class="h-24 text-center">
							{m.no_items_found()}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>
	{#if !withoutPagination}
		{@render Pagination({ table })}
	{/if}
</div>

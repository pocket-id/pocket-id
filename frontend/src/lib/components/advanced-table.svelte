<script lang="ts" generics="T extends {id:string}">
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import * as Pagination from '$lib/components/ui/pagination';
	import * as Select from '$lib/components/ui/select';
	import * as Table from '$lib/components/ui/table/index.js';
	import Empty from '$lib/icons/empty.svelte';
	import { m } from '$lib/paraglide/messages';
	import type {
		Paginated,
		SearchPaginationSortRequest,
		SortRequest
	} from '$lib/types/pagination.type';
	import { debounced } from '$lib/utils/debounce-util';
	import { cn } from '$lib/utils/style';
	import { ChevronDown } from '@lucide/svelte';
	import { PersistedState } from 'runed';
	import { onMount, type Snippet } from 'svelte';
	import Button from './ui/button/button.svelte';
	import { Skeleton } from './ui/skeleton';

	let {
		id,
		selectedIds = $bindable(),
		withoutSearch = false,
		selectionDisabled = false,
		fetchCallback,
		defaultSort,
		columns,
		rows
	}: {
		id: string;
		selectedIds?: string[];
		withoutSearch?: boolean;
		selectionDisabled?: boolean;
		fetchCallback: (requestOptions: SearchPaginationSortRequest) => Promise<Paginated<T>>;
		defaultSort?: SortRequest;
		columns: { label: string; hidden?: boolean; sortColumn?: string }[];
		rows: Snippet<[{ item: T }]>;
	} = $props();

	let items: Paginated<T> | undefined = $state();

	const paginationLimits = new PersistedState<Record<string, number>>('pagination-limits', {});

	const requestOptions = $state<SearchPaginationSortRequest>({
		sort: defaultSort,
		pagination: { limit: 20, page: 1 }
	});

	onMount(async () => {
		if (paginationLimits.current[id]) {
			requestOptions.pagination!.limit = paginationLimits.current[id];
		}
		const urlParams = new URLSearchParams(window.location.search);
		const page = parseInt(urlParams.get(`${id}-page`) ?? '') || undefined;
		if (page) {
			requestOptions.pagination!.page = page;
		}
		await refresh();
	});

	let searchValue = $state('');
	let availablePageSizes: number[] = [20, 50, 100];

	let allChecked = $derived.by(() => {
		if (!selectedIds || !items || items.data.length === 0) return false;
		for (const item of items!.data) {
			if (!selectedIds.includes(item.id)) {
				return false;
			}
		}
		return true;
	});

	const onSearch = debounced(async (search: string) => {
		requestOptions.search = search;
		await refresh();
		searchValue = search;
	}, 300);

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
		paginationLimits.current[id] = size;
		await refresh();
	}

	async function onSort(column?: string, direction: 'asc' | 'desc' = 'asc') {
		if (!column) return;

		requestOptions.sort = { column, direction };
		await refresh();
	}

	function changePageState(page: number) {
		const url = new URL(window.location.href);
		url.searchParams.set(`${id}-page`, page.toString());
		history.replaceState(history.state, '', url.toString());
		requestOptions.pagination!.page = page;
	}

	export async function refresh() {
		items = await fetchCallback(requestOptions);
		changePageState(items.pagination.currentPage);
	}
</script>

{#if !withoutSearch}
	<Input
		value={searchValue}
		class={cn(
			'relative z-50 mb-4 max-w-sm',
			items?.data.length == 0 && searchValue == '' && 'hidden'
		)}
		placeholder={m.search()}
		type="text"
		oninput={(e: Event) => onSearch((e.currentTarget as HTMLInputElement).value)}
	/>
{/if}

{#if items?.pagination.totalItems === 0 && searchValue === ''}
	<div class="my-5 flex flex-col items-center">
		<Empty class="text-muted-foreground h-20" />
		<p class="text-muted-foreground mt-3 text-sm">{m.no_items_found()}</p>
	</div>
{:else}
	<Table.Root
		class="min-w-full overflow-x-auto {items?.data?.length != 0 ? 'table-auto' : 'table-fixed'}"
	>
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

				{#each columns as column}
					<Table.Head class={cn(column.hidden && 'sr-only', column.sortColumn && 'px-0')}>
						{#if column.sortColumn}
							<Button
								variant="ghost"
								class="flex items-center"
								onclick={() =>
									onSort(
										column.sortColumn,
										requestOptions.sort?.direction === 'desc' ? 'asc' : 'desc'
									)}
							>
								{column.label}
								{#if requestOptions.sort?.column === column.sortColumn}
									<ChevronDown
										class={cn(
											'ml-2 size-4',
											requestOptions.sort?.direction === 'asc' ? 'rotate-180' : ''
										)}
									/>
								{/if}
							</Button>
						{:else}
							{column.label}
						{/if}
					</Table.Head>
				{/each}
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#if !items}
				{#each Array(10) as _}
					<tr>
						<td colspan={columns.length + (selectedIds ? 1 : 0)}>
							<Skeleton class="mt-3 h-[40px] w-full rounded-lg" />
						</td>
					</tr>
				{/each}
			{:else}
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
						{@render rows({ item })}
					</Table.Row>
				{/each}
			{/if}
		</Table.Body>
	</Table.Root>

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

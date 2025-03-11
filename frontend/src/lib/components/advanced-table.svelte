<script lang="ts" generics="T extends {id:string}">
	import Checkbox from '$lib/components/ui/checkbox/checkbox.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import * as Pagination from '$lib/components/ui/pagination';
	import * as Select from '$lib/components/ui/select';
	import * as Table from '$lib/components/ui/table/index.js';
	import Empty from '$lib/icons/empty.svelte';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import { debounced } from '$lib/utils/debounce-util';
	import { cn } from '$lib/utils/style';
	import { ChevronDown } from 'lucide-svelte';
	import type { Snippet } from 'svelte';
	import Button from './ui/button/button.svelte';

	let {
		items,
		requestOptions = $bindable(),
		selectedIds = $bindable(),
		withoutSearch = false,
		selectionDisabled = false,
		onRefresh,
		columns,
		rows
	}: {
		items: Paginated<T>;
		requestOptions: SearchPaginationSortRequest;
		selectedIds?: string[];
		withoutSearch?: boolean;
		selectionDisabled?: boolean;
		onRefresh: (requestOptions: SearchPaginationSortRequest) => Promise<Paginated<T>>;
		columns: { label: string; hidden?: boolean; sortColumn?: string }[];
		rows: Snippet<[{ item: T }]>;
	} = $props();

	let searchValue = $state('');
	let availablePageSizes: number[] = [20, 50, 100];

	let allChecked = $derived.by(() => {
		if (!selectedIds || items.data.length === 0) return false;
		for (const item of items.data) {
			if (!selectedIds.includes(item.id)) {
				return false;
			}
		}
		return true;
	});

	const onSearch = debounced(async (search: string) => {
		requestOptions.search = search;
		await onRefresh(requestOptions);
		searchValue = search;
	}, 300);

	async function onAllCheck(checked: boolean) {
		if (checked) {
			selectedIds = items.data.map((item) => item.id);
		} else {
			selectedIds = [];
		}
	}

	async function onCheck(checked: boolean, id: string) {
		if (!selectedIds) return;
		if (checked) {
			selectedIds = [...selectedIds, id];
		} else {
			selectedIds = selectedIds.filter((selectedId) => selectedId !== id);
		}
	}

	async function onPageChange(page: number) {
		requestOptions.pagination = { limit: items.pagination.itemsPerPage, page };
		onRefresh(requestOptions);
	}

	async function onPageSizeChange(size: number) {
		requestOptions.pagination = { limit: size, page: 1 };
		onRefresh(requestOptions);
	}

	async function onSort(column?: string, direction: 'asc' | 'desc' = 'asc') {
		if (!column) return;

		requestOptions.sort = { column, direction };
		onRefresh(requestOptions);
	}
</script>

{#if !withoutSearch}
	<Input
		value={searchValue}
		class={cn(
			'relative z-50 mb-4 max-w-sm',
			items.data.length == 0 && searchValue == '' && 'hidden'
		)}
		placeholder={'Search...'}
		type="text"
		oninput={(e) => onSearch((e.target as HTMLInputElement).value)}
	/>
{/if}

{#if items.data.length === 0 && searchValue === ''}
	<div class="my-5 flex flex-col items-center">
		<Empty class="text-muted-foreground h-20" />
		<p class="text-muted-foreground mt-3 text-sm">No items found</p>
	</div>
{:else}
	<Table.Root class="min-w-full table-auto overflow-x-auto">
		<Table.Header>
			<Table.Row>
				{#if selectedIds}
					<Table.Head class="w-12">
						<Checkbox
							disabled={selectionDisabled}
							checked={allChecked}
							onCheckedChange={(c) => onAllCheck(c as boolean)}
						/>
					</Table.Head>
				{/if}
				{#each columns as column}
					<Table.Head class={cn(column.hidden && 'sr-only', column.sortColumn && 'px-0')}>
						{#if column.sortColumn}
							<Button
								variant="ghost"
								class="flex items-center"
								on:click={() =>
									onSort(
										column.sortColumn,
										requestOptions.sort?.direction === 'desc' ? 'asc' : 'desc'
									)}
							>
								{column.label}
								{#if requestOptions.sort?.column === column.sortColumn}
									<ChevronDown
										class={cn(
											'ml-2 h-4 w-4',
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
			{#each items.data as item}
				<Table.Row class={selectedIds?.includes(item.id) ? 'bg-muted/20' : ''}>
					{#if selectedIds}
						<Table.Cell class="w-12">
							<Checkbox
								disabled={selectionDisabled}
								checked={selectedIds.includes(item.id)}
								onCheckedChange={(c) => onCheck(c as boolean, item.id)}
							/>
						</Table.Cell>
					{/if}
					{@render rows({ item })}
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>

	<div class="mt-5 flex flex-col-reverse items-center justify-between gap-3 sm:flex-row">
		<div class="flex items-center space-x-2">
			<p class="text-sm font-medium">Items per page</p>
			<Select.Root
				selected={{
					label: items.pagination.itemsPerPage.toString(),
					value: items.pagination.itemsPerPage
				}}
				onSelectedChange={(v) => onPageSizeChange(v?.value as number)}
			>
				<Select.Trigger class="h-9 w-[80px]">
					<Select.Value>{items.pagination.itemsPerPage}</Select.Value>
				</Select.Trigger>
				<Select.Content>
					{#each availablePageSizes as size}
						<Select.Item value={size}>{size}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Pagination.Root
			class="mx-0 w-auto"
			count={items.pagination.totalItems}
			perPage={items.pagination.itemsPerPage}
			{onPageChange}
			page={items.pagination.currentPage}
			let:pages
		>
			<Pagination.Content class="flex justify-end">
				<Pagination.Item>
					<Pagination.PrevButton />
				</Pagination.Item>
				{#each pages as page (page.key)}
					{#if page.type !== 'ellipsis' && page.value != 0}
						<Pagination.Item>
							<Pagination.Link {page} isActive={items.pagination.currentPage === page.value}>
								{page.value}
							</Pagination.Link>
						</Pagination.Item>
					{/if}
				{/each}
				<Pagination.Item>
					<Pagination.NextButton />
				</Pagination.Item>
			</Pagination.Content>
		</Pagination.Root>
	</div>
{/if}

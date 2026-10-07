<script lang="ts">
	import * as Pagination from '#lib/components/ui/pagination/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { PaginationResponse } from '#lib/types/list-request.type.ts';
	import { cn } from '#lib/utils/style.ts';

	let {
		pagination,
		onPageChange,
		onPageSizeChange,
		pageSizes = [20, 50, 100],
		hideWhenSinglePage = false,
		class: className
	}: {
		pagination?: PaginationResponse;
		onPageChange: (page: number) => void;
		onPageSizeChange: (size: number) => void;
		pageSizes?: number[];
		hideWhenSinglePage?: boolean;
		class?: string;
	} = $props();

	// The controls are useless when every item already fits on a page of the smallest size
	const hidden = $derived(
		hideWhenSinglePage && (pagination?.totalItems ?? 0) <= Math.min(...pageSizes)
	);
</script>

{#if !hidden}
	<div
		class={cn(
			'mt-5 flex flex-col-reverse items-center justify-between gap-3 sm:flex-row',
			className
		)}
	>
		<div class="flex items-center space-x-2">
			<p class="text-sm font-medium">{m.items_per_page()}</p>
			<Select.Root
				type="single"
				value={pagination?.itemsPerPage.toString()}
				onValueChange={(v) => onPageSizeChange(Number(v))}
			>
				<Select.Trigger class="w-20" aria-label={m.items_per_page()}>
					{pagination?.itemsPerPage}
				</Select.Trigger>
				<Select.Content>
					{#each pageSizes as size (size)}
						<Select.Item value={size.toString()}>{size}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<Pagination.Root
			class="mx-0 w-auto"
			count={pagination?.totalItems || 0}
			perPage={pagination?.itemsPerPage}
			{onPageChange}
			page={pagination?.currentPage}
		>
			{#snippet children({ pages })}
				<Pagination.Content class="flex justify-end">
					<Pagination.Item>
						<Pagination.PrevButton />
					</Pagination.Item>
					{#each pages as page (page.key)}
						{#if page.type !== 'ellipsis' && page.value != 0}
							<Pagination.Item>
								<Pagination.Link {page} isActive={pagination?.currentPage === page.value}>
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

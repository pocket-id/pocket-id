<script lang="ts">
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Select from '#lib/components/ui/select/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import type { PaginationResponse } from '#lib/types/list-request.type.ts';
	import { cn } from '#lib/utils/style.ts';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';

	let {
		pagination,
		onPageChange,
		onPageSizeChange,
		pageSizes = [20, 50, 100],
		class: className
	}: {
		pagination?: PaginationResponse;
		onPageChange: (page: number) => void;
		onPageSizeChange: (size: number) => void;
		pageSizes?: number[];
		class?: string;
	} = $props();

	const totalItems = $derived(pagination?.totalItems ?? 0);
	const currentPage = $derived(pagination?.currentPage ?? 1);
	const perPage = $derived(pagination?.itemsPerPage ?? pageSizes[0]);
	const totalPages = $derived(pagination?.totalPages ?? 1);
	const from = $derived(totalItems ? (currentPage - 1) * perPage + 1 : 0);
	const to = $derived(Math.min(totalItems, currentPage * perPage));

	const fitsSmallestPage = $derived(totalItems <= Math.min(...pageSizes));
</script>

{#if pagination && !fitsSmallestPage}
	<nav
		aria-label={m.pagination()}
		class={cn('text-control flex min-h-(--row) items-center gap-3 pt-2', className)}
	>
		<span class="text-fg-muted flex min-w-0 flex-wrap items-center gap-1">
			<span role="status" class="[&_b]:text-fg pr-1 tabular-nums [&_b]:font-medium">
				<FormattedMessage message={m.pagination_range} inputs={{ from, to, total: totalItems }} />
			</span>
			<span class="flex items-center gap-1 max-sm:hidden">
				<span aria-hidden="true">·</span>
				<Select.Root
					type="single"
					value={String(perPage)}
					onValueChange={(v) => onPageSizeChange(Number(v))}
				>
					<Select.Trigger variant="ghost" class="text-fg-muted" aria-label={m.page_size()}>
						{m.per_page({ count: perPage })}
					</Select.Trigger>
					<Select.Content>
						{#each pageSizes as size (size)}
							<Select.Item value={String(size)} label={m.per_page({ count: size })} />
						{/each}
					</Select.Content>
				</Select.Root>
			</span>
		</span>
		{#if totalPages > 1}
			<span class="ml-auto flex gap-2">
				<Button
					variant="secondary"
					size="sm"
					class="max-sm:px-2"
					disabled={currentPage <= 1}
					onclick={() => onPageChange(currentPage - 1)}
				>
					<ChevronLeftIcon />
					<span class="max-sm:sr-only">{m.previous()}</span>
				</Button>
				<Button
					variant="secondary"
					size="sm"
					class="max-sm:px-2"
					disabled={currentPage >= totalPages}
					onclick={() => onPageChange(currentPage + 1)}
				>
					<span class="max-sm:sr-only">{m.next()}</span>
					<ChevronRightIcon />
				</Button>
			</span>
		{/if}
	</nav>
{/if}

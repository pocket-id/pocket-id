<script lang="ts" generics="TData extends Record<string, any>">
	import { Button, buttonVariants } from '#lib/components/ui/button/index.js';
	import * as Popover from '#lib/components/ui/popover/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import CheckIcon from '@lucide/svelte/icons/check';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';

	let {
		columns,
		selectedColumns = $bindable([])
	}: { columns: AdvancedTableColumn<TData>[]; selectedColumns: string[] } = $props();

	const labelId = $props.id();

	const columnKey = (column: AdvancedTableColumn<TData>) => column.column ?? column.key!;

	function toggle(key: string) {
		selectedColumns = selectedColumns.includes(key)
			? selectedColumns.filter((c) => c !== key)
			: [...selectedColumns, key];
	}
</script>

<Popover.Root>
	<Popover.Trigger
		class={buttonVariants({ variant: 'ghost', size: 'sm', class: 'px-2.5 max-sm:hidden' })}
	>
		<SlidersHorizontalIcon />
		{m.display()}
	</Popover.Trigger>
	<Popover.Content align="end" class="w-70 gap-2">
		<p id={labelId} class="text-caption text-fg-muted font-medium">
			{m.columns()}
		</p>
		<div role="group" aria-labelledby={labelId} class="flex flex-wrap gap-1.5">
			<!-- The first column names the row, so it always shows and has no chip -->
			{#each columns.slice(1).filter((c) => c.label) as column (columnKey(column))}
				{@const pressed = selectedColumns.includes(columnKey(column))}
				<button
					type="button"
					aria-pressed={pressed}
					class="text-control text-fg-muted hover:bg-fill-hover aria-pressed:bg-fill-selected aria-pressed:text-fg focus-visible:outline-ring transition-motion inline-flex h-(--h-sm) items-center gap-1 rounded-sm px-2 shadow-[inset_0_0_0_1px_var(--border-strong)] focus-visible:outline-2 focus-visible:outline-offset-2 active:scale-(--press-scale) aria-pressed:font-medium aria-pressed:shadow-none"
					onclick={() => toggle(columnKey(column))}
				>
					{#if pressed}
						<CheckIcon class="size-3" />
					{/if}
					{column.label}
				</button>
			{/each}
		</div>
		<div class="flex justify-end">
			<Button
				variant="link"
				class="text-caption text-fg-muted"
				onclick={() => (selectedColumns = columns.filter((c) => !c.hidden).map(columnKey))}
			>
				{m.reset()}
			</Button>
		</div>
	</Popover.Content>
</Popover.Root>

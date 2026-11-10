<script lang="ts" generics="TData extends Record<string, any>">
	import { Button, buttonVariants } from '@pocket-id/ui/button';
	import * as Popover from '@pocket-id/ui/popover';
	import { Switch } from '@pocket-id/ui/switch';
	import { cn } from '@pocket-id/ui/utils';
	import { m } from '#lib/paraglide/messages.js';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import { cssMs } from '#lib/utils/style.ts';
	import ReorderIcon from '~icons/solar/reorder-linear';
	import SlidersHorizontalIcon from '~icons/solar/tuning-2-bold';
	import { tick } from 'svelte';

	let {
		columns,
		selectedColumns = $bindable([]),
		columnOrder = $bindable()
	}: {
		columns: AdvancedTableColumn<TData>[];
		selectedColumns: string[];
		columnOrder?: string[];
	} = $props();

	const labelId = $props.id();

	const columnKey = (column: AdvancedTableColumn<TData>) => column.column ?? column.key!;

	// The first column names the row, so it always shows and keeps its place
	const rows = $derived(columns.slice(1).filter((c) => c.label));

	let drag = $state<{
		key: string;
		from: number;
		startY: number;
		rowHeight: number;
		offset: number;
		dropping: boolean;
	} | null>(null);
	const dragTarget = $derived(drag ? drag.from + Math.round(drag.offset / drag.rowHeight) : -1);

	function toggle(key: string) {
		selectedColumns = selectedColumns.includes(key)
			? selectedColumns.filter((c) => c !== key)
			: [...selectedColumns, key];
	}

	function move(key: string, target: string) {
		const keys = columns.slice(1).map(columnKey);
		const targetIndex = keys.indexOf(target);
		keys.splice(keys.indexOf(key), 1);
		keys.splice(targetIndex, 0, key);
		columnOrder = keys;
	}

	// The dragged row follows the pointer while the rows it passes slide over to make room
	function rowOffset(index: number) {
		if (!drag) return 0;
		if (index === drag.from) return drag.offset;
		if (index > drag.from && index <= dragTarget) return -drag.rowHeight;
		if (index < drag.from && index >= dragTarget) return drag.rowHeight;
		return 0;
	}

	function startDrag(e: PointerEvent, key: string, from: number) {
		if (e.button !== 0 || drag) return;
		const grip = e.currentTarget as HTMLElement;
		grip.setPointerCapture(e.pointerId);
		const rowHeight = grip.closest('li')!.offsetHeight;
		drag = { key, from, startY: e.clientY, rowHeight, offset: 0, dropping: false };
	}

	function moveDrag(e: PointerEvent) {
		if (!drag || drag.dropping) return;
		const min = -drag.from * drag.rowHeight;
		const max = (rows.length - 1 - drag.from) * drag.rowHeight;
		drag.offset = Math.min(Math.max(e.clientY - drag.startY, min), max);
	}

	// The row glides into its slot before the order changes, so the list never jumps
	async function endDrag() {
		if (!drag || drag.dropping) return;
		const { key, from } = drag;
		const target = dragTarget;
		drag.offset = (target - from) * drag.rowHeight;
		drag.dropping = true;
		await new Promise((resolve) => setTimeout(resolve, cssMs('--dur-base')));
		if (target !== from) move(key, columnKey(rows[target]));
		drag = null;
	}

	async function moveWithKeyboard(e: KeyboardEvent, key: string, index: number) {
		const step = e.key === 'ArrowUp' ? -1 : e.key === 'ArrowDown' ? 1 : 0;
		const target = rows[index + step];
		if (!step || !target || drag) return;
		e.preventDefault();
		const grip = e.currentTarget as HTMLElement;
		move(key, columnKey(target));
		// Svelte moves the row's node to reorder the list, which drops its focus
		await tick();
		grip.focus();
	}
</script>

<Popover.Root>
	<Popover.Trigger
		aria-label={m.columns()}
		class={buttonVariants({ variant: 'ghost', size: 'sm', class: 'px-2.5 max-sm:hidden' })}
	>
		<SlidersHorizontalIcon />
	</Popover.Trigger>
	<Popover.Content align="end" class="gap-2">
		<p id={labelId} class="text-caption text-fg-muted font-medium">
			{m.columns()}
		</p>
		<ul aria-labelledby={labelId} class={cn('-mx-1.5 select-none', drag && 'cursor-grabbing')}>
			{#each rows as column, i (columnKey(column))}
				{@const key = columnKey(column)}
				{@const dragged = drag?.key === key}
				<li
					class={cn(
						'bg-surface-raised flex h-10 items-center gap-1 rounded-sm pr-1.5',
						drag &&
							(!dragged || drag.dropping) &&
							'transition-transform duration-(--dur-base) ease-(--ease-out)',
						dragged && 'shadow-popover relative z-1'
					)}
					style:transform={drag ? `translateY(${rowOffset(i)}px)` : undefined}
				>
					<button
						type="button"
						aria-label={m.reorder_name({ name: column.label })}
						aria-keyshortcuts="ArrowUp ArrowDown"
						class={cn(
							'text-fg-subtle hover:text-fg focus-visible:outline-ring flex size-8 shrink-0 touch-none items-center justify-center rounded-xs transition-colors duration-(--dur-fast) focus-visible:outline-2',
							drag ? 'cursor-grabbing' : 'cursor-grab',
							dragged && 'text-fg'
						)}
						onpointerdown={(e) => startDrag(e, key, i)}
						onpointermove={moveDrag}
						onpointerup={endDrag}
						onpointercancel={endDrag}
						onkeydown={(e) => moveWithKeyboard(e, key, i)}
					>
						<ReorderIcon class="size-4" />
					</button>
					<label
						class="text-control flex min-w-0 flex-1 cursor-pointer items-center justify-between gap-3"
					>
						<span class="truncate">{column.label}</span>
						<Switch
							size="sm"
							checked={selectedColumns.includes(key)}
							onCheckedChange={() => toggle(key)}
						/>
					</label>
				</li>
			{/each}
		</ul>
		<div class="flex justify-end">
			<Button
				variant="link"
				class="text-caption text-fg-muted"
				onclick={() => {
					selectedColumns = columns.filter((c) => !c.hidden).map(columnKey);
					columnOrder = undefined;
				}}
			>
				{m.reset()}
			</Button>
		</div>
	</Popover.Content>
</Popover.Root>

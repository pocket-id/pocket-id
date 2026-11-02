<script lang="ts">
	import { Badge } from '@pocket-id/ui/badge';
	import * as Command from '@pocket-id/ui/command';
	import { filledFieldClass } from '@pocket-id/ui/input';
	import * as Popover from '@pocket-id/ui/popover';
	import { Spinner } from '@pocket-id/ui/spinner';
	import { m } from '#lib/paraglide/messages.js';
	import { searchThreshold } from '#lib/utils/select-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { LucideChevronDown } from '@lucide/svelte';
	import type { FormEventHandler } from 'svelte/elements';

	let {
		id,
		items,
		selectedItems = $bindable(),
		oninput,
		isLoading = false,
		placeholder = m.select_an_option()
	}: {
		id?: string;
		items: {
			value: string;
			label: string;
		}[];
		selectedItems: string[];
		// Searching on the server, so the items are only a page of the results and the search field always shows
		oninput?: FormEventHandler<HTMLInputElement>;
		isLoading?: boolean;
		placeholder?: string;
	} = $props();

	let open = $state(false);
	let commandRef = $state<HTMLElement | null>(null);
	let inputRef = $state<HTMLElement | null>(null);
	let search = $state('');
	const searchable = $derived(!!oninput || items.length > searchThreshold);
	const visibleItems = $derived(
		oninput
			? items
			: items.filter((item) => item.label.toLowerCase().includes(search.toLowerCase()))
	);
	const selected = $derived(items.filter((item) => selectedItems.includes(item.value)));

	let badgeRow = $state<HTMLElement>();
	let measureRow = $state<HTMLElement>();
	let visibleCount = $state(Infinity);
	const hiddenCount = $derived(Math.max(0, selected.length - visibleCount));

	// The measure row grows with every selected badge, so observing it also refits after a selection change
	$effect(() => {
		if (!badgeRow || !measureRow) return;

		const row = badgeRow;
		const measure = measureRow;
		const observer = new ResizeObserver(() => fitBadges(row, measure));
		observer.observe(row);
		observer.observe(measure);

		return () => observer.disconnect();
	});

	// Shows as many badges as fit on one line and leaves room for the counter when some have to be hidden
	function fitBadges(row: HTMLElement, measure: HTMLElement) {
		const badges = [...measure.children] as HTMLElement[];
		const counter = badges.pop()!;
		const fitsWithin = (width: number) =>
			badges.filter((badge) => badge.offsetLeft + badge.offsetWidth <= width).length;

		if (fitsWithin(row.clientWidth) === badges.length) {
			visibleCount = badges.length;
			return;
		}

		const gap = parseFloat(getComputedStyle(measure).columnGap);
		visibleCount = fitsWithin(row.clientWidth - counter.offsetWidth - gap);
	}

	function toggleItem(value: string) {
		if (selectedItems.includes(value)) {
			selectedItems = selectedItems.filter((item) => item !== value);
		} else {
			selectedItems = [...selectedItems, value];
		}
	}
</script>

<Popover.Root bind:open onOpenChange={(isOpen) => isOpen && (search = '')}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				{id}
				type="button"
				role="combobox"
				aria-expanded={open}
				class={cn(
					filledFieldClass,
					'text-control aria-expanded:bg-fill-selected dark:aria-expanded:bg-fill-selected flex h-(--h-md) w-full items-center justify-between gap-1.5 rounded-sm pr-2.5 pl-3.5 text-left'
				)}
			>
				<!-- The badges stay on one line so the trigger keeps the height of the other inputs, and the inline-size containment stops them from widening the field -->
				<span
					bind:this={badgeRow}
					class="relative flex min-w-0 flex-1 gap-1 overflow-hidden contain-inline-size"
				>
					{#each selected.slice(0, visibleCount) as item (item.value)}
						<Badge variant="outline" class="bg-surface">{item.label}</Badge>
					{:else}
						{#if hiddenCount === 0}
							<span class="text-fg-muted truncate">{placeholder}</span>
						{/if}
					{/each}
					{#if hiddenCount > 0}
						<Badge variant="outline" class="bg-surface">+{hiddenCount}</Badge>
					{/if}
					<span
						bind:this={measureRow}
						aria-hidden="true"
						class="invisible absolute flex w-max gap-1"
					>
						{#each selected as item (item.value)}
							<Badge variant="outline">{item.label}</Badge>
						{/each}
						<Badge variant="outline">+{selected.length}</Badge>
					</span>
				</span>
				<LucideChevronDown class="text-fg-muted size-3.5 shrink-0" />
			</button>
		{/snippet}
	</Popover.Trigger>
	<!-- The content is focused before it is positioned, so a scrolling focus would make the page jump -->
	<!-- Without a search field nothing in the list is tabbable, so the command root takes focus to keep arrow keys and Enter working -->
	<Popover.Content
		class="p-0"
		sameWidth
		onOpenAutoFocus={(e) => {
			e.preventDefault();
			(inputRef ?? commandRef)?.focus({ preventScroll: true });
		}}
	>
		<Command.Root bind:ref={commandRef} shouldFilter={false} class="outline-hidden">
			{#if searchable}
				<Command.Input bind:ref={inputRef} bind:value={search} placeholder={m.search()} {oninput} />
				<Command.Empty>
					{#if isLoading}
						<div class="flex w-full justify-center">
							<Spinner />
						</div>
					{:else}
						{m.no_items_found()}
					{/if}
				</Command.Empty>
			{/if}
			<Command.Group class="max-h-60 overflow-y-auto">
				{#each visibleItems as item (item.value)}
					<Command.Item
						aria-checked={selectedItems.includes(item.value)}
						data-checked={selectedItems.includes(item.value)}
						value={item.value}
						onSelect={() => toggleItem(item.value)}
					>
						<span class="min-w-0 wrap-anywhere">{item.label}</span>
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

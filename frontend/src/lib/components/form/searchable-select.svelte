<script lang="ts">
	import * as Command from '@pocket-id/ui/command';
	import { filledFieldClass } from '@pocket-id/ui/input';
	import * as Popover from '@pocket-id/ui/popover';
	import { Spinner } from '@pocket-id/ui/spinner';
	import { m } from '#lib/paraglide/messages.js';
	import { searchThreshold } from '#lib/utils/select-util.ts';
	import { cn } from '@pocket-id/ui/utils';
	import { LucideChevronDown } from '@lucide/svelte';
	import type { FormEventHandler, HTMLButtonAttributes } from 'svelte/elements';

	let {
		items,
		value = $bindable(),
		onSelect,
		oninput,
		isLoading,
		selectText = m.select_an_option(),
		class: className,
		...restProps
	}: Omit<HTMLButtonAttributes, 'value' | 'oninput'> & {
		items: {
			value: string;
			label: string;
			description?: string;
		}[];
		value: string;
		// Searching on the server, so the items are only a page of the results and the search field always shows
		oninput?: FormEventHandler<HTMLInputElement>;
		onSelect?: (value: string) => void;
		isLoading?: boolean;
		selectText?: string;
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
	const selectedLabel = $derived(items.find((item) => item.value === value)?.label);
</script>

<Popover.Root bind:open onOpenChange={(isOpen) => isOpen && (search = '')}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				{...restProps}
				type="button"
				role="combobox"
				aria-expanded={open}
				class={cn(
					filledFieldClass,
					'text-control aria-expanded:bg-fill-selected dark:aria-expanded:bg-fill-selected flex h-(--h-md) w-full items-center justify-between gap-1.5 rounded-sm pr-2.5 pl-3.5 text-left',
					className
				)}
			>
				<span class={cn('truncate', !selectedLabel && 'text-fg-muted')}>
					{selectedLabel || selectText}
				</span>
				<LucideChevronDown class="text-fg-muted size-3.5 shrink-0" />
			</button>
		{/snippet}
	</Popover.Trigger>
	<!-- The content is focused before it is positioned, so a scrolling focus would make the page jump -->
	<!-- Without a search field nothing in the list is tabbable, so the command root takes focus to keep arrow keys and Enter working -->
	<Popover.Content
		class="min-w-56 p-0"
		sameWidth
		align="end"
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
						value={item.value}
						data-checked={value === item.value}
						onSelect={() => {
							value = item.value;
							onSelect?.(item.value);
							open = false;
						}}
					>
						{#if item.description === undefined}
							{item.label}
						{:else}
							<span class="flex min-w-0 flex-col">
								<span class="truncate">{item.label}</span>
								<span class="text-caption text-fg-muted truncate">{item.description}</span>
							</span>
						{/if}
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

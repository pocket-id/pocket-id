<script lang="ts">
	import * as Command from '#lib/components/ui/command/index.ts';
	import { filledFieldClass } from '#lib/components/ui/input/index.ts';
	import * as Popover from '#lib/components/ui/popover/index.ts';
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideCheck, LucideChevronDown } from '@lucide/svelte';
	import type { FormEventHandler, HTMLAttributes } from 'svelte/elements';

	let {
		items,
		value = $bindable(),
		onSelect,
		oninput,
		isLoading,
		disableSearch = false,
		selectText = m.select_an_option(),
		...restProps
	}: HTMLAttributes<HTMLButtonElement> & {
		items: {
			value: string;
			label: string;
		}[];
		value: string;
		oninput?: FormEventHandler<HTMLInputElement>;
		onSelect?: (value: string) => void;
		isLoading?: boolean;
		disableSearch?: boolean;
		selectText?: string;
	} = $props();

	let open = $state(false);
	let filteredItems = $state(items);
	const selectedLabel = $derived(items.find((item) => item.value === value)?.label);

	function filterItems(searchString: string) {
		if (!searchString) {
			filteredItems = items;
		} else {
			filteredItems = items.filter((item) =>
				item.label.toLowerCase().includes(searchString.toLowerCase())
			);
		}
	}

	// Reset items when opening again
	$effect(() => {
		if (open) {
			filteredItems = items;
		}
	});
</script>

<Popover.Root bind:open {...restProps}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				role="combobox"
				aria-expanded={open}
				class={cn(
					filledFieldClass,
					'text-control aria-expanded:bg-fill-selected dark:aria-expanded:bg-fill-selected flex h-(--h-md) w-full items-center justify-between gap-1.5 rounded-sm pr-2.5 pl-3.5 text-left',
					restProps.class
				)}
			>
				<span class={cn('truncate', !selectedLabel && 'text-fg-muted')}>
					{selectedLabel || selectText}
				</span>
				<LucideChevronDown class="text-fg-muted size-3.5 shrink-0" />
			</button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="p-0" sameWidth>
		<Command.Root shouldFilter={false}>
			{#if !disableSearch}
				<Command.Input
					placeholder={m.search()}
					oninput={(e) => {
						filterItems(e.currentTarget.value);
						oninput?.(e);
					}}
				/>
			{/if}
			<Command.Empty>
				{#if isLoading}
					<div class="flex w-full justify-center">
						<Spinner />
					</div>
				{:else}
					{m.no_items_found()}
				{/if}
			</Command.Empty>
			<Command.Group>
				{#each filteredItems as item (item.value)}
					<Command.Item
						value={item.value}
						onSelect={() => {
							value = item.value;
							onSelect?.(item.value);
							open = false;
						}}
					>
						<LucideCheck class={cn('size-4', value !== item.value && 'text-transparent')} />
						{item.label}
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Command from '#lib/components/ui/command/index.ts';
	import * as Popover from '#lib/components/ui/popover/index.ts';
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideCheck, LucideChevronDown } from '@lucide/svelte';
	import { tick } from 'svelte';
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

	// We want to refocus the trigger button when the user selects
	// an item from the list so users can continue navigating the
	// rest of the form with the keyboard.
	function closeAndFocusTrigger(triggerId: string) {
		open = false;
		tick().then(() => {
			document.getElementById(triggerId)?.focus();
		});
	}

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
			<Button
				variant="outline"
				role="combobox"
				aria-expanded={open}
				{...props}
				class={restProps.class}
			>
				<span class="flex w-full justify-between">
					{items.find((item) => item.value === value)?.label || selectText}
					<LucideChevronDown class="size-4 shrink-0 opacity-50" />
				</span>
			</Button>
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
							// If you need to focus the trigger, you may need to refactor to get the trigger id another way
							closeAndFocusTrigger('popover-trigger');
						}}
					>
						<LucideCheck class={cn('mr-2 size-4', value !== item.value && 'text-transparent')} />
						{item.label}
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

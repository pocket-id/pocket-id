<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Command from '$lib/components/ui/command';
	import * as Popover from '$lib/components/ui/popover';
	import { cn } from '$lib/utils/style';
	import { LucideCheck, LucideChevronDown } from 'lucide-svelte';
	import { tick } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		items,
		value = $bindable(),
		onSelect,
		...restProps
	}: HTMLAttributes<HTMLButtonElement> & {
		items: {
			value: string;
			label: string;
		}[];
		value: string;
		onSelect?: (value: string) => void;
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

<Popover.Root bind:open let:ids>
	<Popover.Trigger asChild let:builder>
		<Button
			{...restProps}
			builders={[builder]}
			variant="outline"
			role="combobox"
			aria-expanded={open}
			class={cn('justify-between', restProps.class)}
		>
			{items.find((item) => item.value === value)?.label || 'Select an option'}
			<LucideChevronDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
		</Button>
	</Popover.Trigger>
	<Popover.Content class="p-0" sameWidth>
		<Command.Root shouldFilter={false}>
			<Command.Input placeholder="Search..." oninput={(e: any) => filterItems(e.target.value)} />
			<Command.Empty>No results found.</Command.Empty>
			<Command.Group>
				{#each filteredItems as item}
					<Command.Item
						value={item.value}
						onSelect={() => {
							value = item.value;
							onSelect?.(item.value);
							closeAndFocusTrigger(ids.trigger);
						}}
					>
						<LucideCheck class={cn('mr-2 h-4 w-4', value !== item.value && 'text-transparent')} />
						{item.label}
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

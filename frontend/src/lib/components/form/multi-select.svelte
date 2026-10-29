<script lang="ts">
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideChevronDown } from '@lucide/svelte';
	import { Badge } from '../ui/badge';
	import { filledFieldClass } from '../ui/input';

	let {
		items,
		selectedItems = $bindable(),
		onSelect,
		autoClose = false,
		placeholder = m.select_an_option(),
		class: className
	}: {
		items: {
			value: string;
			label: string;
		}[];
		selectedItems: string[];
		onSelect?: (value: string) => void;
		autoClose?: boolean;
		placeholder?: string;
		class?: string;
	} = $props();

	const selected = $derived(items.filter((item) => selectedItems.includes(item.value)));

	function handleItemSelect(value: string) {
		if (selectedItems.includes(value)) {
			selectedItems = selectedItems.filter((item) => item !== value);
		} else {
			selectedItems = [...selectedItems, value];
		}
		onSelect?.(value);
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				class={cn(
					filledFieldClass,
					'text-control aria-expanded:bg-fill-selected dark:aria-expanded:bg-fill-selected flex h-(--h-md) w-full items-center justify-between gap-1.5 rounded-sm pr-2.5 pl-3.5 text-left',
					className
				)}
			>
				<!-- The badges stay on one line and are clipped so the trigger keeps the height of the other inputs -->
				<span class="flex min-w-0 gap-1 overflow-hidden">
					{#each selected as item (item.value)}
						<Badge variant="outline" class="bg-surface">{item.label}</Badge>
					{:else}
						<span class="text-fg-muted truncate">{placeholder}</span>
					{/each}
				</span>
				<LucideChevronDown class="text-fg-muted size-3.5 shrink-0" />
			</button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-[var(--bits-dropdown-menu-anchor-width)]">
		{#each items as item (item.value)}
			<DropdownMenu.CheckboxItem
				checked={selectedItems.includes(item.value)}
				onCheckedChange={() => handleItemSelect(item.value)}
				closeOnSelect={autoClose}
			>
				{item.label}
			</DropdownMenu.CheckboxItem>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>

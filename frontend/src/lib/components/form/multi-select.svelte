<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { LucideChevronDown } from '@lucide/svelte';
	import { Badge } from '../ui/badge';
	import { Button } from '../ui/button';

	let {
		items,
		selectedItems = $bindable(),
		onSelect,
		autoClose = false
	}: {
		items: {
			value: string;
			label: string;
		}[];
		selectedItems: string[];
		onSelect?: (value: string) => void;
		autoClose?: boolean;
	} = $props();

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
			<Button {...props} variant="outline">
				{#each items.filter((item) => selectedItems.includes(item.value)) as item}
					<Badge variant="secondary">
						{item.label}
					</Badge>
				{/each}
				<LucideChevronDown class="text-muted-foreground ml-2 size-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-[var(--bits-dropdown-menu-anchor-width)]">
		{#each items as item}
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

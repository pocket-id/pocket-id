<script lang="ts">
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideChevronDown } from '@lucide/svelte';
	import { Badge } from '../ui/badge';
	import { Button } from '../ui/button';

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
			<Button {...props} variant="outline" class={cn('w-full px-3 font-normal', className)}>
				<!-- Button centers its content in an inner span, so this one spreads the badges and the chevron apart -->
				<span class="flex w-full min-w-0 items-center justify-between gap-2">
					<!-- The badges stay on one line and are clipped so the trigger keeps the height of the other inputs -->
					<span class="flex min-w-0 gap-1 overflow-hidden">
						{#each selected as item (item.value)}
							<Badge variant="secondary">{item.label}</Badge>
						{:else}
							<span class="text-muted-foreground">{placeholder}</span>
						{/each}
					</span>
					<LucideChevronDown class="size-4 shrink-0 opacity-50" />
				</span>
			</Button>
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

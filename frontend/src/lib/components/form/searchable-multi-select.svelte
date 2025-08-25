<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Command from '$lib/components/ui/command';
	import * as Popover from '$lib/components/ui/popover';
	import { cn } from '$lib/utils/style';
    import { m } from '$lib/paraglide/messages';
	import { LoaderCircle, LucideCheck, LucideChevronDown } from '@lucide/svelte';
	import type { FormEventHandler } from 'svelte/elements';

	type Item = {
		value: string;
		label: string;
	};

	let {
		items,
		selectedItems = $bindable(),
		onSelect,
		oninput,
		isLoading = false,
		disableInternalSearch = false,
		id
	}: {
		items: Item[];
		selectedItems: string[];
		onSelect?: (value: string[]) => void;
		oninput?: FormEventHandler<HTMLInputElement>;
		isLoading?: boolean;
		disableInternalSearch?: boolean;
		id?: string;
	} = $props();

	let open = $state(false);
	let searchValue = $state('');
	let filteredItems = $state(items);

	const selectedLabels = $derived(
		items.filter((item) => selectedItems.includes(item.value)).map((item) => item.label)
	);

	function handleItemSelect(value: string) {
		let newSelectedItems: string[];
		if (selectedItems.includes(value)) {
			newSelectedItems = selectedItems.filter((item) => item !== value);
		} else {
			newSelectedItems = [...selectedItems, value];
		}
		selectedItems = newSelectedItems;
		onSelect?.(newSelectedItems);
	}

	function filterItems(search: string) {
		if (disableInternalSearch) return;
		searchValue = search;
		if (!search) {
			filteredItems = items;
		} else {
			filteredItems = items.filter((item) =>
				item.label.toLowerCase().includes(search.toLowerCase())
			);
		}
	}

	// Reset search value when the popover is closed
	$effect(() => {
		if (!open) {
			filterItems('');
		}

		filteredItems = items;
	});
</script>

<Popover.Root bind:open>
	<Popover.Trigger {id}>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="outline"
				role="combobox"
				aria-expanded={open}
				class="h-auto min-h-10 w-full justify-between"
			>
				<div class="flex flex-wrap items-center gap-1">
					{#if selectedItems.length > 0}
						{#each selectedLabels as label}
							<Badge variant="secondary">{label}</Badge>
						{/each}
					{:else}
						<span class="text-muted-foreground font-normal">{m.select_items()}</span>
					{/if}
				</div>
				<LucideChevronDown class="ml-2 size-4 shrink-0 opacity-50" />
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="p-0" sameWidth>
		<Command.Root shouldFilter={false}>
			<Command.Input
				placeholder={m.search()}
				value={searchValue}
				oninput={(e) => {
					filterItems(e.currentTarget.value);
					oninput?.(e);
				}}
			/>
			<Command.Empty>
				{#if isLoading}
					<div class="flex w-full items-center justify-center py-2">
						<LoaderCircle class="size-4 animate-spin" />
					</div>
				{:else}
					{m.no_items_found()}
				{/if}
			</Command.Empty>
			<Command.Group class="max-h-60 overflow-y-auto">
				{#each filteredItems as item}
					<Command.Item
						aria-checked={selectedItems.includes(item.value)}
						value={item.value}
						onSelect={() => {
							handleItemSelect(item.value);
						}}
					>
						<LucideCheck
							class={cn('mr-2 size-4', !selectedItems.includes(item.value) && 'text-transparent')}
						/>
						{item.label}
					</Command.Item>
				{/each}
			</Command.Group>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

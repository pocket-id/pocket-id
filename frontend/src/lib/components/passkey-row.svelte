<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Tooltip from '$lib/components/ui/tooltip/index.js';
	import { m } from '$lib/paraglide/messages';
	import { LucideCalendar, LucidePencil, LucideTrash, type Icon as IconType } from '@lucide/svelte';

	let {
		icon,
		onRename,
		onDelete,
		label,
		description
	}: {
		icon: typeof IconType;
		onRename: () => void;
		onDelete: () => void;
		description?: string;
		label?: string;
	} = $props();
</script>

<Item.Root variant="transparent" class="hover:bg-muted transition-colors">
	<Item.Media class="bg-primary/10 text-primary rounded-lg p-2">
		{#if icon}{@const Icon = icon}
			<Icon class="size-5" />
		{/if}
	</Item.Media>
	<Item.Content>
		<Item.Title>{label}</Item.Title>
		{#if description}
			<Item.Description class="flex items-center">
				<LucideCalendar class="mr-1 size-3" />
				{description}
			</Item.Description>
		{/if}
	</Item.Content>
	<Item.Actions>
		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<Button
						onclick={onRename}
						size="icon"
						variant="ghost"
						class="size-8"
						aria-label={m.rename()}
					>
						<LucidePencil class="size-4" />
					</Button>
				</Tooltip.Trigger>
				<Tooltip.Content>{m.rename()}</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>

		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<Button
						onclick={onDelete}
						size="icon"
						variant="ghost"
						class="hover:bg-destructive/10 hover:text-destructive size-8"
						aria-label={m.delete()}
					>
						<LucideTrash class="size-4" />
					</Button>
				</Tooltip.Trigger>
				<Tooltip.Content>{m.delete()}</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>
	</Item.Actions>
</Item.Root>

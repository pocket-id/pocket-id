<script lang="ts">
	import Checkbox from '#lib/components/ui/checkbox/checkbox.svelte';
	import * as Item from '#lib/components/ui/item/index.js';
	import { cn } from '#lib/utils/style.js';
	import type { LucideIcon } from '@lucide/svelte';

	interface Props {
		icon: LucideIcon;
		name: string;
		description: string;
		selected?: boolean;
		onSelectedChange?: (selected: boolean) => void;
	}

	let { icon, name, description, selected = true, onSelectedChange }: Props = $props();

	const SvelteComponent = $derived(icon);
</script>

{#snippet content()}
	<Item.Media class="bg-muted h-full !translate-y-0 !self-center rounded-lg p-2">
		<SvelteComponent class="size-5" />
	</Item.Media>
	<Item.Content class="text-start">
		<Item.Title class="font-semibold">{name}</Item.Title>
		<Item.Description>{description}</Item.Description>
	</Item.Content>
{/snippet}

{#if onSelectedChange}
	<!-- The whole row is a label so clicking anywhere on it toggles the scope -->
	<Item.Root class="px-0 py-1.5">
		{#snippet child({ props })}
			<label
				{...props}
				class={cn(props.class as string, 'cursor-pointer', !selected && 'opacity-60')}
				data-testid="scope-item"
			>
				{@render content()}
				<Item.Actions>
					<Checkbox
						aria-label={name}
						checked={selected}
						onCheckedChange={(checked: boolean) => onSelectedChange(checked)}
					/>
				</Item.Actions>
			</label>
		{/snippet}
	</Item.Root>
{:else}
	<Item.Root class="px-0 py-1.5">
		{@render content()}
	</Item.Root>
{/if}

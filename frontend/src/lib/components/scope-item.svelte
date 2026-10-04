<script lang="ts">
	import * as Item from '#lib/components/ui/item/index.js';
	import type { LucideIcon } from '@lucide/svelte';

	interface Props {
		icon: LucideIcon;
		name: string;
		description: string;
	}

	let { icon, name, description }: Props = $props();

	const SvelteComponent = $derived(icon);
</script>

<Item.Root class="px-0 py-1.5">
	<Item.Media class="bg-muted h-full !translate-y-0 !self-center rounded-lg p-2">
		<SvelteComponent class="size-5" />
	</Item.Media>
	<Item.Content class="text-start">
		<Item.Title class="font-semibold">{name}</Item.Title>
		<Item.Description>{description}</Item.Description>
	</Item.Content>
</Item.Root>

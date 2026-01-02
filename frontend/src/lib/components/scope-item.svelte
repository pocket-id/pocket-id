<script lang="ts">
	import * as Item from '$lib/components/ui/item/index.js';
	import type { Icon as IconType } from '@lucide/svelte';

	interface Props {
		icon: typeof IconType;
		name: string;
		description: string;
	}

	let { icon, name, description }: Props = $props();

	const SvelteComponent = $derived(icon);
</script>

<Item.Root class="py-1.5 px-0">
	<Item.Media class="bg-muted !self-center rounded-lg p-2 !translate-y-0 h-full">
		<SvelteComponent class="size-5" />
	</Item.Media>
	<Item.Content class="text-start">
		<Item.Title class="font-semibold">{name}</Item.Title>
		<Item.Description>{description}</Item.Description>
	</Item.Content>
</Item.Root>

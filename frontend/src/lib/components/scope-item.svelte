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

<Item.Root size="sm">
	<Item.Media variant="icon">
		<SvelteComponent class="size-4" />
	</Item.Media>
	<Item.Content>
		<Item.Title class="font-medium">{name}</Item.Title>
		<Item.Description>{description}</Item.Description>
	</Item.Content>
</Item.Root>

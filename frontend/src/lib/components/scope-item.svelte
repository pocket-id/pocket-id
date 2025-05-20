<script lang="ts">
	import type { Icon as IconType } from '@lucide/svelte';
	interface Props {
		icon: typeof IconType;
		name: string;
		description: string;
	}

	let { icon, name, description }: Props = $props();

	const SvelteComponent = $derived(icon);
</script>

<div class="flex items-center">
	<div class="bg-muted mr-5 rounded-lg p-2"><SvelteComponent /></div>
	<div class="text-start">
		<h3 class="font-semibold">{name}</h3>
		<p class="text-muted-foreground text-sm">{description}</p>
	</div>
</div>

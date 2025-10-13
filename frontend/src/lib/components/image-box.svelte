<script lang="ts">
	import { cn } from '$lib/utils/style';
	import { LucideImageOff } from '@lucide/svelte';
	import type { HTMLImgAttributes } from 'svelte/elements';

	let props: HTMLImgAttributes & {} = $props();
	let error = $state(false);

	$effect(() => {
		props.src;
		error = false;
	});
</script>

<div class={cn('bg-muted flex items-center justify-center rounded-2xl p-3', props.class)}>
	{#if error}
		<LucideImageOff class="text-muted-foreground p-5" />
	{:else}
		<img {...props} class="aspect-square object-contain" onerror={() => (error = true)} />
	{/if}
</div>

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

<div class={'bg-muted flex items-center justify-center rounded-2xl p-3'}>
	{#if error}
		<LucideImageOff class={cn('text-muted-foreground p-5', props.class)} />
	{:else}
		<img
			{...props}
			class={cn('object-contain aspect-square', props.class)}
			onerror={() => (error = true)}
		/>
	{/if}
</div>

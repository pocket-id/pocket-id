<script lang="ts">
	import Logo from '$lib/components/logo.svelte';
	import CheckmarkAnimated from '$lib/icons/checkmark-animated.svelte';
	import CrossAnimated from '$lib/icons/cross-animated.svelte';
	import { fade } from 'svelte/transition';

	const {
		error,
		success
	}: {
		error?: boolean;
		success?: boolean;
	} = $props();
</script>

<div
	class="rounded-2xl p-3 transition-[background-color] duration-300
		 {error ? 'bg-red-200' : success ? 'bg-green-200' : 'bg-muted'}"
>
	{#if error || success}
		<div class="flex size-10 items-center justify-center">
			{#if error}
				<CrossAnimated class="size-5" />
			{:else}
				<CheckmarkAnimated class="size-5" />
			{/if}
		</div>
	{:else}
		<div in:fade={{ duration: 300 }}>
			<Logo class="size-10" />
		</div>
	{/if}
</div>

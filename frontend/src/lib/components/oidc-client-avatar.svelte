<script lang="ts">
	import ImageBox from '$lib/components/image-box.svelte';
	import { m } from '$lib/paraglide/messages';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { cn } from '$lib/utils/style';
	import { mode } from 'mode-watcher';

	let {
		id,
		name,
		hasLogo,
		class: className = 'size-9'
	}: {
		id: string;
		name: string;
		hasLogo: boolean;
		class?: string;
	} = $props();

	const isLightMode = $derived(mode.current === 'light');
</script>

{#if hasLogo}
	<ImageBox
		class={cn('rounded-lg', className)}
		src={cachedOidcClientLogo.getUrl(id, isLightMode)}
		alt={m.name_logo({ name })}
	/>
{:else}
	<div
		class={cn('bg-muted flex shrink-0 items-center justify-center rounded-lg font-bold', className)}
	>
		{name.charAt(0).toUpperCase()}
	</div>
{/if}

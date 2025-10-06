<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { cachedApplicationLogo } from '$lib/utils/cached-image-util';
	import { cn } from '$lib/utils/style';
	import { mode } from 'mode-watcher';
	import type { HTMLAttributes } from 'svelte/elements';

	let { ...props }: HTMLAttributes<HTMLImageElement> = $props();

	const isLightMode = $derived(mode.current === 'light');
</script>

<img
	{...props}
	class={cn('aspect-square object-contain', props.class)}
	src={cachedApplicationLogo.getUrl(isLightMode)}
	alt={m.logo()}
/>

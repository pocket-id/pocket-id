<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { mode } from 'mode-watcher';
	import type { HTMLAttributes } from 'svelte/elements';

	let { ...props }: HTMLAttributes<HTMLImageElement> = $props();

	const isDarkMode = $derived(mode.current === 'dark');
</script>

<img {...props} src="/api/application-configuration/logo?light={!isDarkMode}" alt={m.logo()} />

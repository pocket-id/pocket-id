<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { getLogoUrl } from '$lib/utils/logo-util';
	import type { HTMLAttributes } from 'svelte/elements';

	let { ...props }: HTMLAttributes<HTMLImageElement> = $props();

	const logoUrl = getLogoUrl();
</script>

<img {...props} src={logoUrl} alt={m.logo()} />

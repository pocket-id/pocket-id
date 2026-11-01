<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import { LucideArrowUpRight } from '@lucide/svelte';

	let { href }: { href: string } = $props();
</script>

<a {href} target="_blank" rel="noreferrer" class="inline-link inline-flex items-center gap-1">
	{m.docs()}
	<LucideArrowUpRight class="size-3" />
</a>

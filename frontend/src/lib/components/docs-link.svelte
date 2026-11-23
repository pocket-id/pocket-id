<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import ArrowUpRightIcon from '~icons/solar/arrow-right-up-linear';

	let { href }: { href: string } = $props();
</script>

<a {href} target="_blank" rel="noreferrer" class="inline-link inline-flex items-center gap-1">
	{m.docs()}
	<ArrowUpRightIcon class="size-3" />
</a>

<script lang="ts">
	import type { Snippet } from 'svelte';

	// A failure replaces the subtitle in the same muted text, because the tile above already turns red
	let { title, children }: { title: string | Snippet; children?: Snippet } = $props();
</script>

<h1 class="font-display text-display-m sm:text-display-l mb-2 wrap-anywhere">
	{#if typeof title === 'string'}
		{title}
	{:else}
		{@render title()}
	{/if}
</h1>
<p class="text-body-l text-fg-muted mb-6 wrap-anywhere" aria-live="polite">
	{@render children?.()}
</p>

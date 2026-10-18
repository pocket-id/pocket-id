<script lang="ts">
	import { cn } from '#lib/utils/style.ts';
	import { LucideCircleAlert, LucideInfo } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		title,
		tone,
		children
	}: {
		title: string | Snippet;
		// A reason replaces the subtitle in the text color, so a failure never shifts what is below it
		tone?: 'info' | 'danger';
		children?: Snippet;
	} = $props();
</script>

<h1 class="font-display text-display-m sm:text-display-l mb-2 wrap-anywhere">
	{#if typeof title === 'string'}
		{title}
	{:else}
		{@render title()}
	{/if}
</h1>
<p
	class={cn('text-body-l mb-6 wrap-anywhere', tone ? 'text-fg' : 'text-fg-muted')}
	aria-live="polite"
>
	{#if tone === 'danger'}
		<LucideCircleAlert class="text-danger-solid mr-1.5 inline size-3.5 align-[-1px]" />
	{:else if tone === 'info'}
		<LucideInfo class="text-fg-muted mr-1.5 inline size-3.5 align-[-1px]" />
	{/if}
	{@render children?.()}
</p>

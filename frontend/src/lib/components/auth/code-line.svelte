<script lang="ts">
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { cn } from '#lib/utils/style.ts';
	import { LucideCircleAlert, LucideInfo } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		id,
		tone,
		children
	}: {
		id: string;
		tone?: 'checking' | 'error' | 'info';
		children?: Snippet;
	} = $props();
</script>

<!-- The line is always present, so a status appearing under the code never moves what is below it -->
<p
	{id}
	aria-live="polite"
	class={cn(
		'text-control mt-2 flex min-h-5 items-start gap-1.5',
		tone === 'error' ? 'text-danger-fg' : tone === 'info' ? 'text-fg' : 'text-fg-muted'
	)}
>
	{#if tone === 'checking'}
		<Spinner class="mt-1 size-3" />
	{:else if tone === 'error'}
		<LucideCircleAlert class="mt-1 size-3 shrink-0" />
	{:else if tone === 'info'}
		<LucideInfo class="mt-1 size-3 shrink-0" />
	{/if}
	<span>{@render children?.()}</span>
</p>

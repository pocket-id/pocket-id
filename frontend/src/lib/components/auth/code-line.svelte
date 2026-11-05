<script lang="ts">
	import { Spinner } from '@pocket-id/ui/spinner';
	import { cn } from '@pocket-id/ui/utils';
	import CircleAlertIcon from '~icons/solar/danger-circle-bold-duotone';
	import type { Snippet } from 'svelte';

	let {
		id,
		tone,
		children
	}: {
		id: string;
		tone?: 'checking' | 'error';
		children?: Snippet;
	} = $props();
</script>

<!-- The line is always present, so a status appearing under the code never moves what is below it -->
<p
	{id}
	aria-live="polite"
	class={cn(
		'text-control mt-2 flex min-h-5 items-start gap-1.5',
		tone === 'error' ? 'text-danger-fg' : 'text-fg-muted'
	)}
>
	{#if tone === 'checking'}
		<Spinner class="mt-1 size-3" />
	{:else if tone === 'error'}
		<CircleAlertIcon class="mt-1 size-3 shrink-0" />
	{/if}
	<span>{@render children?.()}</span>
</p>

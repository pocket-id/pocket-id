<script lang="ts" module>
	// An option tile of a radio group, shared by every section's pickers
	export const tileClass =
		'bg-surface text-fg rounded-md text-left shadow-[inset_0_0_0_1px_var(--border)] transition-shadow duration-(--dur-fast) [&:hover:not([data-state=checked])]:shadow-[inset_0_0_0_1px_var(--border-strong)] data-checked:bg-surface-raised data-checked:shadow-[inset_0_0_0_1.5px_var(--fg),var(--elevation-selected)] focus-visible:outline-ring focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-45';
</script>

<script lang="ts">
	import { cn } from '@pocket-id/ui/utils';
	import type { Snippet } from 'svelte';
	import PreviewFigure from './preview-figure.svelte';

	let {
		title,
		stageClass,
		class: className,
		children,
		preview
	}: {
		title: string;
		stageClass?: string;
		class?: string;
		children: Snippet;
		// Without one the section is a column of controls beside a preview its parent places
		preview?: Snippet;
	} = $props();

	const headingId = $props.id();
</script>

<section
	aria-labelledby={headingId}
	class={cn(
		'grid gap-x-14 gap-y-7 py-12 shadow-[inset_0_1px_0_var(--border)]',
		preview && 'lg:grid-cols-[minmax(0,9fr)_minmax(0,11fr)]',
		className
	)}
>
	<div class="flex min-w-0 flex-col gap-7">
		<h2 id={headingId} class="text-title text-fg">{title}</h2>
		{@render children()}
	</div>

	{#if preview}
		<PreviewFigure {stageClass}>
			{@render preview()}
		</PreviewFigure>
	{/if}
</section>

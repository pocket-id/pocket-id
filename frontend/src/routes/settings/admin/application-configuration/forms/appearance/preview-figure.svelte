<script lang="ts">
	import { previewClass, previewMode } from '#lib/components/appearance/appearance.svelte.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '@pocket-id/ui/utils';
	import { RadioGroup } from 'bits-ui';
	import MoonIcon from '~icons/solar/moon-linear';
	import SunIcon from '~icons/solar/sun-2-linear';
	import type { Snippet } from 'svelte';

	let {
		stageClass,
		class: className,
		children
	}: {
		stageClass?: string;
		class?: string;
		children: Snippet;
	} = $props();

	const modes = [
		{ value: 'light', label: m.light(), icon: SunIcon },
		{ value: 'dark', label: m.dark(), icon: MoonIcon }
	] as const;
</script>

<!-- The preview follows a tall column of controls, so what they change stays in view -->
<figure
	class={cn(
		'border-border bg-surface relative min-w-0 self-start overflow-hidden rounded-xl border lg:sticky lg:top-[calc(var(--clear-top)+var(--spacing)*8)]',
		className
	)}
>
	<!-- The bottom padding keeps the content clear of the mode switch floating in the corner -->
	<div
		inert
		class={cn(
			previewClass(),
			'bg-surface-sunken grid min-h-60 place-items-center px-5 pt-5 pb-14 sm:px-8 sm:pt-8 sm:pb-14',
			stageClass
		)}
	>
		{@render children()}
	</div>
	<RadioGroup.Root
		aria-label="Preview mode"
		orientation="horizontal"
		class="bg-surface-raised/90 border-border shadow-thumb absolute right-3 bottom-3 flex items-center gap-0.5 rounded-full border p-0.5 backdrop-blur-sm"
		bind:value={() => previewMode.current, (v) => (previewMode.current = v as 'light' | 'dark')}
	>
		{#each modes as option (option.value)}
			<RadioGroup.Item
				value={option.value}
				aria-label={option.label}
				title={option.label}
				class="text-fg-muted hover:text-fg data-checked:bg-fill-selected data-checked:text-fg focus-visible:outline-ring grid size-7 place-items-center rounded-full transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-1"
			>
				<option.icon class="size-3.5" />
			</RadioGroup.Item>
		{/each}
	</RadioGroup.Root>
</figure>

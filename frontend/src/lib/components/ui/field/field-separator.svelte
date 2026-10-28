<script lang="ts">
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		children?: Snippet;
	} = $props();

	const hasContent = $derived(!!children);
</script>

<div
	bind:this={ref}
	data-slot="field-separator"
	data-content={hasContent}
	class={cn(
		'text-caption relative -my-2 h-5 group-data-[variant=outline]/field-group:-mb-2',
		className
	)}
	{...restProps}
>
	<Separator class="absolute inset-0 top-1/2" />
	{#if children}
		<span
			class="bg-background text-fg-muted relative mx-auto block w-fit px-2"
			data-slot="field-separator-content"
		>
			{@render children()}
		</span>
	{/if}
</div>

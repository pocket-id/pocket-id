<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		child,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		child?: Snippet<[{ props: Record<string, unknown> }]>;
	} = $props();

	const mergedProps = $derived({
		...restProps,
		class: cn(
			"flex items-center gap-2 rounded-sm bg-fill-muted px-3.5 text-control text-fg-muted dark:bg-fg/6 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4",
			className
		),
		'data-slot': 'button-group-text'
	});
</script>

{#if child}
	{@render child({ props: mergedProps })}
{:else}
	<div bind:this={ref} {...mergedProps}>
		{@render mergedProps.children?.()}
	</div>
{/if}

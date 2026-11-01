<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn, type WithElementRef } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="alert-dialog-media"
	class={cn(
		"bg-fill-muted mb-2 inline-flex size-12 items-center justify-center rounded-full *:[svg:not([class*='size-'])]:size-6",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

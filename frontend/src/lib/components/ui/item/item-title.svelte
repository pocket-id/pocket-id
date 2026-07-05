<script lang="ts">
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="item-title"
	class={cn(
		'gap-2 text-sm leading-snug font-semibold underline-offset-4 line-clamp-1 flex w-fit items-center',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

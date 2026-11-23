<script lang="ts">
	import { cn, type WithElementRef, type WithoutChildren } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildren<WithElementRef<HTMLAttributes<HTMLDivElement>>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="skeleton"
	class={cn(
		'bg-fill-skeleton animate-[pulse_var(--t-pulse)_ease-in-out_infinite] rounded-xs',
		className
	)}
	{...restProps}
></div>

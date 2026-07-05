<script lang="ts">
	import { cn, type WithElementRef, type WithoutChildren } from '$lib/utils/style.js';
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
	class={cn('bg-muted rounded-2xl animate-pulse', className)}
	{...restProps}
></div>

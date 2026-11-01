<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLInputAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		value = $bindable(),
		class: className,
		...restProps
	}: WithElementRef<HTMLInputAttributes> = $props();
</script>

<input
	bind:this={ref}
	bind:value
	data-slot="input-group-control"
	class={cn(
		'text-body text-fg placeholder:text-fg-muted h-full w-full min-w-0 flex-1 bg-transparent px-3.5 outline-hidden max-sm:text-[16px] [&::-webkit-search-cancel-button]:appearance-none',
		className
	)}
	{...restProps}
/>

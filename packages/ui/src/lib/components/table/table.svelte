<script lang="ts">
	import type { HTMLTableAttributes } from 'svelte/elements';
	import { cn, type WithElementRef } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLTableAttributes> = $props();
</script>

<!-- The block padding keeps focus rings in the header row visible inside the horizontal scroller -->
<div data-slot="table-container" class="relative -my-1 w-full overflow-x-auto py-1">
	<table
		bind:this={ref}
		data-slot="table"
		class={cn('text-control w-full caption-bottom', className)}
		{...restProps}
	>
		{@render children?.()}
	</table>
</div>

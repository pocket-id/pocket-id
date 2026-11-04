<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLTableSectionElement>> = $props();
</script>

<!-- The gap below the header is an empty row because collapsed borders let cell fills reach halfway into a border -->
<thead
	bind:this={ref}
	data-slot="table-header"
	class={cn('after:table-row after:h-1', className)}
	{...restProps}
>
	{@render children?.()}
</thead>

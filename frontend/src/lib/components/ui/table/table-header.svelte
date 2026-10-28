<script lang="ts">
	import type { WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLTableSectionElement>> = $props();
</script>

<thead bind:this={ref} data-slot="table-header" class={className} {...restProps}>
	{@render children?.()}
</thead>

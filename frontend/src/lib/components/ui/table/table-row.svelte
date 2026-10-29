<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLTableRowElement>> = $props();
</script>

<tr
	bind:this={ref}
	data-slot="table-row"
	class={cn(
		'in-[tbody]:hover:bg-fill-row-hover in-[tbody]:has-focus-visible:bg-fill-row-hover data-[state=selected]:bg-fill-muted border-b transition-colors duration-(--dur-fast) in-[thead]:border-b-0',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</tr>

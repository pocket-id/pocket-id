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

<!-- Fills sit on the cells so they round like the header, and the dividers around a filled row hide so it reads as one pill -->
<!-- They are inset shadows because a custom appearance makes them translucent, and an opaque sticky cell must still show them over its surface -->
<tr
	bind:this={ref}
	data-slot="table-row"
	class={cn(
		'border-b transition-colors duration-(--dur-fast) in-[tbody]:hover:border-transparent in-[tbody]:hover:*:shadow-[inset_0_0_0_100vmax_var(--fill-row-hover)] in-[thead]:border-b-4 in-[thead]:border-transparent in-[tbody]:has-focus-visible:border-transparent in-[tbody]:has-focus-visible:*:shadow-[inset_0_0_0_100vmax_var(--fill-row-hover)] in-[tbody]:has-[+tr_:focus-visible]:border-transparent in-[tbody]:has-[+tr:hover]:border-transparent data-[state=selected]:*:shadow-[inset_0_0_0_100vmax_var(--fill-muted)]',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</tr>

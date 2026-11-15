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
<!-- Surface bands above and below the fill keep a gap between adjacent filled rows, such as a run of selected ones -->
<tr
	bind:this={ref}
	data-slot="table-row"
	class={cn(
		'border-b transition-colors duration-(--dur-fast) in-[tbody]:*:shadow-[inset_0_1px_var(--surface),inset_0_-1px_var(--surface),inset_0_0_0_100vmax_var(--row-fill,transparent)] in-[tbody]:hover:border-transparent in-[tbody]:hover:[--row-fill:var(--fill-row-hover)] in-[thead]:border-b-0 in-[tbody]:has-focus-visible:border-transparent in-[tbody]:has-focus-visible:[--row-fill:var(--fill-row-hover)] in-[tbody]:has-[+tr_:focus-visible]:border-transparent in-[tbody]:has-[+tr:hover]:border-transparent in-[tbody]:has-[+tr[data-state=selected]]:border-transparent data-[state=selected]:border-transparent data-[state=selected]:[--row-fill:var(--fill-muted)]',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</tr>

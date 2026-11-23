<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
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
	data-slot="empty"
	class={cn(
		'flex w-full max-w-[420px] min-w-0 flex-col items-start gap-4 pt-16 pb-12 text-left',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

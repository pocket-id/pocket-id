<script lang="ts">
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		wrapOnMobile,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		wrapOnMobile?: boolean;
	} = $props();
</script>

<div
	bind:this={ref}
	data-slot="item-actions"
	class={cn(
		'flex items-center gap-2',
		wrapOnMobile && 'w-full pl-8 sm:w-auto sm:pl-0 pt-1',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

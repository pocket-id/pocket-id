<script lang="ts">
	import { cn, type WithElementRef, type WithoutChildren } from '#lib/utils/style.js';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		value = $bindable(),
		class: className,
		...restProps
	}: WithoutChildren<WithElementRef<HTMLTextareaAttributes>> = $props();
</script>

<textarea
	bind:this={ref}
	bind:value
	data-slot="input-group-control"
	class={cn(
		'text-body text-fg placeholder:text-fg-muted min-h-[86px] w-full flex-1 resize-none bg-transparent px-3.5 py-2.5 outline-hidden max-sm:text-[16px]',
		className
	)}
	{...restProps}></textarea>

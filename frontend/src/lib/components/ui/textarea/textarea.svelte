<script lang="ts">
	import { filledFieldClass, readOnlyFieldClass } from '#lib/components/ui/input/index.js';
	import { cn, type WithElementRef, type WithoutChildren } from '#lib/utils/style.js';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		value = $bindable(),
		class: className,
		'data-slot': dataSlot = 'textarea',
		...restProps
	}: WithoutChildren<WithElementRef<HTMLTextareaAttributes>> = $props();
</script>

<!-- min-h-[86px] holds three body lines plus the vertical padding -->
<textarea
	bind:this={ref}
	data-slot={dataSlot}
	class={cn(
		filledFieldClass,
		readOnlyFieldClass,
		'text-body placeholder:text-fg-muted flex min-h-[86px] w-full resize-y rounded-sm px-3.5 py-2.5 max-sm:text-[16px]',
		className
	)}
	bind:value
	{...restProps}></textarea>

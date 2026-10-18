<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import { filledFieldClass } from '#lib/components/ui/input/index.js';
	import { cn, type WithoutChild } from '#lib/utils/style.js';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';

	let {
		ref = $bindable(null),
		class: className,
		children,
		size = 'default',
		...restProps
	}: WithoutChild<SelectPrimitive.TriggerProps> & {
		size?: 'sm' | 'default';
	} = $props();
</script>

<SelectPrimitive.Trigger
	bind:ref
	data-slot="select-trigger"
	data-size={size}
	class={cn(
		filledFieldClass,
		"text-control aria-expanded:bg-fill-selected data-placeholder:text-fg-muted dark:aria-expanded:bg-fill-selected flex w-fit min-w-0 items-center justify-between gap-1.5 rounded-sm text-left whitespace-nowrap data-[size=default]:h-(--h-md) data-[size=default]:pr-2.5 data-[size=default]:pl-3.5 data-[size=sm]:h-(--h-sm) data-[size=sm]:pr-2 data-[size=sm]:pl-3 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		className
	)}
	{...restProps}
>
	{@render children?.()}
	<ChevronDownIcon class="text-fg-muted pointer-events-none size-3.5" />
</SelectPrimitive.Trigger>

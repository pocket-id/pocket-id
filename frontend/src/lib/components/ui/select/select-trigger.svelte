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
		variant = 'filled',
		...restProps
	}: WithoutChild<SelectPrimitive.TriggerProps> & {
		size?: 'sm' | 'default';
		// Ghost triggers read as text inside a sentence or a filter row (facet filters, the page size)
		variant?: 'filled' | 'ghost';
	} = $props();
</script>

<SelectPrimitive.Trigger
	bind:ref
	data-slot="select-trigger"
	data-size={size}
	class={cn(
		"text-control flex w-fit min-w-0 items-center justify-between gap-1.5 rounded-sm text-left whitespace-nowrap [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		variant === 'filled'
			? cn(
					filledFieldClass,
					'aria-expanded:bg-fill-selected data-placeholder:text-fg-muted dark:aria-expanded:bg-fill-selected data-[size=default]:h-(--h-md) data-[size=default]:pr-2.5 data-[size=default]:pl-3.5 data-[size=sm]:h-(--h-sm) data-[size=sm]:pr-2 data-[size=sm]:pl-3'
				)
			: 'hover:bg-fill-hover aria-expanded:bg-fill-selected focus-visible:outline-ring h-7 gap-1 rounded-xs pr-1.5 pl-2 transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-45',
		className
	)}
	{...restProps}
>
	{@render children?.()}
	<ChevronDownIcon
		class={cn('text-fg-muted pointer-events-none', variant === 'ghost' ? 'size-3' : 'size-3.5')}
	/>
</SelectPrimitive.Trigger>

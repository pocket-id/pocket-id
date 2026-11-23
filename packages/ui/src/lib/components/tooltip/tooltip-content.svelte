<script lang="ts">
	import { Tooltip as TooltipPrimitive } from 'bits-ui';
	import { cn } from '#lib/utils/style.js';
	import TooltipPortal from './tooltip-portal.svelte';
	import type { ComponentProps } from 'svelte';
	import type { WithoutChildrenOrChild } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		sideOffset = 6,
		side = 'top',
		portalProps,
		...restProps
	}: TooltipPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof TooltipPortal>>;
	} = $props();
</script>

<TooltipPortal {...portalProps}>
	<TooltipPrimitive.Content
		bind:ref
		data-slot="tooltip-content"
		{sideOffset}
		{side}
		class={cn(
			'bg-inverse text-inverse-fg inset-ring-inverse-border text-caption transition-motion z-(--z-tooltip) inline-flex w-fit max-w-60 items-center gap-1.5 rounded-sm px-2 py-1 inset-ring data-closed:opacity-0 data-closed:ease-in starting:opacity-0',
			className
		)}
		{...restProps}
	/>
</TooltipPortal>

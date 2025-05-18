<script lang="ts">
	import { Tooltip as TooltipPrimitive } from 'bits-ui';
	import { cn, flyAndScale } from '$lib/utils/style.js';

	type $$Props = TooltipPrimitive.ContentProps;

	interface Props {
		class?: $$Props['class'];
		sideOffset?: $$Props['sideOffset'];
		transition?: $$Props['transition'];
		transitionConfig?: $$Props['transitionConfig'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let {
		class: className = undefined,
		sideOffset = 4,
		transition = flyAndScale,
		transitionConfig = {
		y: 8,
		duration: 150
	},
		children,
		...rest
	}: Props = $props();
	
</script>

<TooltipPrimitive.Content
	{transition}
	{transitionConfig}
	{sideOffset}
	class={cn(
		'bg-popover text-popover-foreground z-50 overflow-hidden rounded-md border px-3 py-1.5 text-sm shadow-md',
		className
	)}
	{...rest}
>
	{@render children?.()}
</TooltipPrimitive.Content>

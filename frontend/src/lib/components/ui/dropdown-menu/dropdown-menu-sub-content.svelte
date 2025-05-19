<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import { cn, flyAndScale } from '$lib/utils/style.js';

	type $$Props = DropdownMenuPrimitive.SubContentProps;
	type $$Events = DropdownMenuPrimitive.SubContentEvents;

	interface Props {
		class?: $$Props['class'];
		transition?: $$Props['transition'];
		transitionConfig?: $$Props['transitionConfig'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let {
		class: className = undefined,
		transition = flyAndScale,
		transitionConfig = {
		x: -10,
		y: 0
	},
		children,
		...rest
	}: Props = $props();
	
</script>

<DropdownMenuPrimitive.SubContent
	{transition}
	{transitionConfig}
	class={cn(
		'bg-popover text-popover-foreground z-50 min-w-[8rem] rounded-md border p-1 shadow-lg focus:outline-none',
		className
	)}
	{...rest}
	on:keydown
	on:focusout
	on:pointermove
>
	{@render children?.()}
</DropdownMenuPrimitive.SubContent>

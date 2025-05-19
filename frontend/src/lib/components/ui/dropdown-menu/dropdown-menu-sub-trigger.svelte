<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';
	import { cn } from '$lib/utils/style.js';

	type $$Props = DropdownMenuPrimitive.SubTriggerProps & {
		inset?: boolean;
	};
	type $$Events = DropdownMenuPrimitive.SubTriggerEvents;

	interface Props {
		class?: $$Props['class'];
		inset?: $$Props['inset'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, inset = undefined, children, ...rest }: Props = $props();
	
</script>

<DropdownMenuPrimitive.SubTrigger
	class={cn(
		'data-[highlighted]:bg-accent data-[state=open]:bg-accent data-[highlighted]:text-accent-foreground data-[state=open]:text-accent-foreground flex items-center rounded-sm px-2 py-1.5 text-sm outline-none select-none',
		inset && 'pl-8',
		className
	)}
	{...rest}
	on:click
	on:keydown
	on:focusin
	on:focusout
	on:pointerleave
	on:pointermove
>
	{@render children?.()}
	<ChevronRight class="ml-auto h-4 w-4" />
</DropdownMenuPrimitive.SubTrigger>

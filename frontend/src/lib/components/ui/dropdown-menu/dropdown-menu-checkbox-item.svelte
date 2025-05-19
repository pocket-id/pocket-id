<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import Check from 'lucide-svelte/icons/check';
	import { cn } from '$lib/utils/style.js';

	type $$Props = DropdownMenuPrimitive.CheckboxItemProps;
	type $$Events = DropdownMenuPrimitive.CheckboxItemEvents;

	interface Props {
		class?: $$Props['class'];
		checked?: $$Props['checked'];
		children?: import('svelte').Snippet;
		[key: string]: any
	}

	let { class: className = undefined, checked = $bindable(undefined), children, ...rest }: Props = $props();
	
</script>

<DropdownMenuPrimitive.CheckboxItem
	bind:checked
	class={cn(
		'data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground relative flex cursor-default items-center rounded-sm py-1.5 pr-2 pl-8 text-sm outline-none select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
		className
	)}
	{...rest}
	on:click
	on:keydown
	on:focusin
	on:focusout
	on:pointerdown
	on:pointerleave
	on:pointermove
>
	<span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
		<DropdownMenuPrimitive.CheckboxIndicator>
			<Check class="h-4 w-4" />
		</DropdownMenuPrimitive.CheckboxIndicator>
	</span>
	{@render children?.()}
</DropdownMenuPrimitive.CheckboxItem>

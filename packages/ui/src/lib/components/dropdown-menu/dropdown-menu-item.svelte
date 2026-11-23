<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';

	let {
		ref = $bindable(null),
		class: className,
		inset,
		variant = 'default',
		...restProps
	}: DropdownMenuPrimitive.ItemProps & {
		inset?: boolean;
		variant?: 'default' | 'destructive';
	} = $props();
</script>

<DropdownMenuPrimitive.Item
	bind:ref
	data-slot="dropdown-menu-item"
	data-inset={inset}
	data-variant={variant}
	class={cn(
		"group/dropdown-menu-item data-highlighted:bg-fg/6 text-control not-data-[variant=destructive]:[&_svg:not([class*='text-'])]:text-fg-muted data-[variant=destructive]:text-danger-fg data-[variant=destructive]:data-highlighted:bg-danger-soft data-[variant=destructive]:*:[svg]:text-danger-fg relative flex min-h-9 cursor-pointer items-center gap-2 rounded-sm px-2.5 outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-45 data-inset:pl-8.5 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		className
	)}
	{...restProps}
/>

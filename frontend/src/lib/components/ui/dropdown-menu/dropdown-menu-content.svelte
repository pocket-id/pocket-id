<script lang="ts">
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import type { ComponentProps } from 'svelte';
	import DropdownMenuPortal from './dropdown-menu-portal.svelte';

	let {
		ref = $bindable(null),
		sideOffset = 4,
		align = 'start',
		portalProps,
		class: className,
		...restProps
	}: DropdownMenuPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof DropdownMenuPortal>>;
	} = $props();
</script>

<DropdownMenuPortal {...portalProps}>
	<DropdownMenuPrimitive.Content
		bind:ref
		data-slot="dropdown-menu-content"
		{sideOffset}
		{align}
		class={cn(
			'data-open:animate-in data-closed:animate-out data-closed:fade-out-0 data-open:fade-in-0 data-closed:zoom-out-95 data-open:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 ring-foreground/5 dark:ring-foreground/10 text-popover-foreground data-[side=inline-start]:slide-in-from-right-2 data-[side=inline-end]:slide-in-from-left-2 bg-popover/70 **:data-[slot$=-item]:focus:bg-foreground/10 **:data-[slot$=-item]:data-highlighted:bg-foreground/10 **:data-[slot$=-separator]:bg-foreground/5 **:data-[slot$=-trigger]:focus:bg-foreground/10 **:data-[slot$=-trigger]:aria-expanded:bg-foreground/10! **:data-[variant=destructive]:data-highlighted:bg-destructive/10! dark:**:data-[variant=destructive]:data-highlighted:bg-destructive/20! relative z-50 min-w-48 animate-none! overflow-x-hidden overflow-y-auto rounded-3xl p-1.5 shadow-lg ring-1 duration-100 outline-none before:pointer-events-none before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:backdrop-blur-2xl before:backdrop-saturate-150 data-closed:overflow-hidden',
			className
		)}
		{...restProps}
	/>
</DropdownMenuPortal>

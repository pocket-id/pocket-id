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
			'text-fg shadow-popover bg-surface-raised/72 border-fg/6 dark:border-fg/10 [@media(prefers-reduced-transparency:reduce)]:bg-surface-raised! transition-motion z-(--z-popover) max-h-(--bits-dropdown-menu-content-available-height) min-w-50 origin-(--bits-dropdown-menu-content-transform-origin) overflow-x-hidden overflow-y-auto rounded-md border p-1 backdrop-blur-[40px] backdrop-saturate-150 duration-(--dur-base) outline-none data-closed:pointer-events-none data-closed:opacity-0 data-closed:duration-(--dur-fast) data-closed:ease-in starting:scale-(--pop-scale) starting:opacity-0 [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
			className
		)}
		{...restProps}
	/>
</DropdownMenuPortal>

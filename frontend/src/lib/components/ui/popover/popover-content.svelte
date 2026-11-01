<script lang="ts">
	import { Popover as PopoverPrimitive } from 'bits-ui';
	import PopoverPortal from './popover-portal.svelte';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		sideOffset = 4,
		align = 'center',
		sameWidth = false,
		portalProps,
		...restProps
	}: PopoverPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof PopoverPortal>>;
		sameWidth?: boolean;
	} = $props();
</script>

<PopoverPortal {...portalProps}>
	<PopoverPrimitive.Content
		bind:ref
		data-slot="popover-content"
		{sideOffset}
		{align}
		class={cn(
			'bg-popover text-popover-foreground border-border shadow-popover transition-motion has-data-[slot=command]:bg-popover/72 has-data-[slot=command]:border-fg/6 dark:has-data-[slot=command]:border-fg/10 [@media(prefers-reduced-transparency:reduce)]:bg-popover! z-(--z-popover) flex w-72 origin-(--bits-popover-content-transform-origin) flex-col gap-4 rounded-md border p-3 outline-hidden duration-(--dur-base) has-data-[slot=command]:backdrop-blur-[40px] has-data-[slot=command]:backdrop-saturate-150 data-closed:pointer-events-none data-closed:opacity-0 data-closed:duration-(--dur-fast) data-closed:ease-in starting:scale-(--pop-scale) starting:opacity-0 [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
			sameWidth && 'w-(--bits-popover-anchor-width)',
			className
		)}
		{...restProps}
	/>
</PopoverPortal>

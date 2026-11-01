<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import SelectPortal from './select-portal.svelte';
	import SelectScrollUpButton from './select-scroll-up-button.svelte';
	import SelectScrollDownButton from './select-scroll-down-button.svelte';
	import { cn, type WithoutChild } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';
	import type { WithoutChildrenOrChild } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		sideOffset = 4,
		collisionPadding = 8,
		portalProps,
		children,
		preventScroll = true,
		...restProps
	}: WithoutChild<SelectPrimitive.ContentProps> & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof SelectPortal>>;
	} = $props();
</script>

<SelectPortal {...portalProps}>
	<SelectPrimitive.Content
		bind:ref
		{sideOffset}
		{collisionPadding}
		{preventScroll}
		data-slot="select-content"
		class={cn(
			'text-popover-foreground shadow-popover bg-popover/72 border-fg/6 dark:border-fg/10 [@media(prefers-reduced-transparency:reduce)]:bg-popover! transition-motion **:data-[slot=select-item]:data-highlighted:bg-fg/6 relative z-(--z-popover) max-h-(--bits-select-content-available-height) max-w-(--bits-select-content-available-width) min-w-36 origin-(--bits-select-content-transform-origin) overflow-x-hidden overflow-y-auto rounded-md border p-1 backdrop-blur-[40px] backdrop-saturate-150 duration-(--dur-base) outline-none data-closed:pointer-events-none data-closed:opacity-0 data-closed:duration-(--dur-fast) data-closed:ease-in starting:scale-(--pop-scale) starting:opacity-0 [@media(prefers-reduced-transparency:reduce)]:backdrop-filter-none!',
			className
		)}
		{...restProps}
	>
		<SelectScrollUpButton />
		<SelectPrimitive.Viewport
			class={cn(
				'h-(--bits-select-anchor-height) w-full min-w-(--bits-select-anchor-width) scroll-my-1'
			)}
		>
			{@render children?.()}
		</SelectPrimitive.Viewport>
		<SelectScrollDownButton />
	</SelectPrimitive.Content>
</SelectPortal>

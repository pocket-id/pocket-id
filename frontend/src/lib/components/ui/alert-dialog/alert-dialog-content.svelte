<script lang="ts">
	import { AlertDialog as AlertDialogPrimitive } from 'bits-ui';
	import AlertDialogPortal from './alert-dialog-portal.svelte';
	import AlertDialogOverlay from './alert-dialog-overlay.svelte';
	import { cn, type WithoutChild, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		size = 'default',
		portalProps,
		...restProps
	}: WithoutChild<AlertDialogPrimitive.ContentProps> & {
		size?: 'default' | 'sm';
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof AlertDialogPortal>>;
	} = $props();
</script>

<AlertDialogPortal {...portalProps}>
	<AlertDialogOverlay />
	<AlertDialogPrimitive.Content
		bind:ref
		data-slot="alert-dialog-content"
		data-size={size}
		class={cn(
			'group/alert-dialog-content bg-popover text-popover-foreground border-border shadow-overlay transition-motion fixed top-1/2 left-1/2 z-(--z-overlay) grid max-h-[85dvh] w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 gap-4 overflow-y-auto rounded-xl border px-6 py-5 duration-(--dur-slow) outline-none data-closed:opacity-0 data-closed:duration-(--dur-base) data-closed:ease-in sm:data-[size=default]:max-w-(--dialog-sm) sm:data-[size=sm]:max-w-xs starting:scale-(--pop-scale) starting:opacity-0',
			'max-sm:before:bg-border-strong max-sm:inset-x-0 max-sm:top-auto max-sm:bottom-0 max-sm:w-full max-sm:translate-0 max-sm:rounded-t-lg max-sm:rounded-b-none max-sm:pt-6 max-sm:pb-[calc(20px+env(safe-area-inset-bottom))] max-sm:before:absolute max-sm:before:top-2 max-sm:before:left-1/2 max-sm:before:h-1 max-sm:before:w-9 max-sm:before:-translate-x-1/2 max-sm:before:rounded-[2px] max-sm:data-closed:translate-y-full max-sm:data-closed:opacity-100 max-sm:starting:translate-y-full max-sm:starting:scale-100 max-sm:starting:opacity-100',
			className
		)}
		{...restProps}
	/>
</AlertDialogPortal>

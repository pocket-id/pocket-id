<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import DialogPortal from './dialog-portal.svelte';
	import type { Snippet } from 'svelte';
	import * as Dialog from './index.js';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import XIcon from '@lucide/svelte/icons/x';
	import { m } from '#lib/paraglide/messages.js';

	let {
		ref = $bindable(null),
		class: className,
		portalProps,
		overlayClass,
		children,
		showCloseButton = true,
		...restProps
	}: WithoutChildrenOrChild<DialogPrimitive.ContentProps> & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof DialogPortal>>;
		overlayClass?: string;
		children: Snippet;
		showCloseButton?: boolean;
	} = $props();
</script>

<DialogPortal {...portalProps}>
	<Dialog.Overlay class={overlayClass} />
	<!-- The phone bottom sheet is a set of max-sm overrides, so the centered position and width callers pass in class never reach it -->
	<DialogPrimitive.Content
		bind:ref
		data-slot="dialog-content"
		class={cn(
			'bg-popover text-popover-foreground border-border shadow-overlay transition-motion fixed top-1/2 left-1/2 z-(--z-overlay) grid max-h-[85dvh] w-[calc(100%-2rem)] max-w-(--dialog-md) -translate-x-1/2 -translate-y-1/2 gap-4 overflow-y-auto rounded-xl border px-6 py-5 duration-(--dur-slow) outline-none data-closed:opacity-0 data-closed:duration-(--dur-base) data-closed:ease-in has-[>[data-slot=dialog-close]]:*:data-[slot=dialog-header]:pr-6 starting:scale-(--pop-scale) starting:opacity-0',
			'max-sm:before:bg-border-strong max-sm:inset-x-0 max-sm:top-auto max-sm:bottom-0 max-sm:w-full max-sm:max-w-none max-sm:translate-0 max-sm:rounded-t-lg max-sm:rounded-b-none max-sm:pt-6 max-sm:pb-[calc(20px+env(safe-area-inset-bottom))] max-sm:before:absolute max-sm:before:top-2 max-sm:before:left-1/2 max-sm:before:h-1 max-sm:before:w-9 max-sm:before:-translate-x-1/2 max-sm:before:rounded-[2px] max-sm:data-closed:translate-y-full max-sm:data-closed:opacity-100 max-sm:starting:translate-y-full max-sm:starting:scale-100 max-sm:starting:opacity-100',
			className
		)}
		{...restProps}
	>
		{@render children?.()}
		{#if showCloseButton}
			<DialogPrimitive.Close data-slot="dialog-close">
				{#snippet child({ props })}
					<Button
						variant="ghost"
						class="absolute top-3.5 right-3.5 size-7"
						size="icon-sm"
						{...props}
					>
						<XIcon />
						<span class="sr-only">{m.close()}</span>
					</Button>
				{/snippet}
			</DialogPrimitive.Close>
		{/if}
	</DialogPrimitive.Content>
</DialogPortal>

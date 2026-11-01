<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import DialogOverlay from '#lib/components/ui/dialog/dialog-overlay.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import XIcon from '@lucide/svelte/icons/x';
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		title,
		footer,
		class: className,
		children,
		...restProps
	}: WithoutChildrenOrChild<DialogPrimitive.ContentProps> & {
		title: string;
		// Sticky actions under the scrolling body, right-aligned: the ghost Cancel, then the primary
		footer?: Snippet;
		children: Snippet;
	} = $props();

	const FIELD =
		'input:not([type="hidden"]):not(:disabled), textarea:not(:disabled), [role="combobox"], [role="radio"]';

	// Desktop starts in the first field so people can type right away; phones focus the sheet itself so the keyboard doesn't cover it
	function focusFirstField(event: Event) {
		event.preventDefault();
		const phone = window.matchMedia('(width < 40rem)').matches;
		const field = phone ? null : ref?.querySelector<HTMLElement>(FIELD);
		(field ?? ref)?.focus();
	}
</script>

<DialogPrimitive.Portal>
	<DialogOverlay />
	<!-- Inset from the window like the panel, and a full-height bottom sheet on phones -->
	<DialogPrimitive.Content
		bind:ref
		data-slot="sheet-content"
		class={cn(
			'bg-surface-raised text-fg shadow-overlay transition-motion fixed z-(--z-overlay) flex flex-col overflow-hidden border outline-none',
			'inset-x-0 top-2 bottom-0 rounded-t-lg duration-(--dur-slow) data-closed:translate-y-full data-closed:duration-(--dur-base) data-closed:ease-in starting:translate-y-full',
			'sm:inset-y-2 sm:right-2 sm:left-auto sm:w-[min(var(--sheet-w),calc(100%-48px))] sm:rounded-lg sm:data-closed:translate-x-(--slide) sm:data-closed:translate-y-0 sm:data-closed:opacity-0 sm:starting:translate-x-(--slide) sm:starting:translate-y-0 sm:starting:opacity-0',
			className
		)}
		onOpenAutoFocus={focusFirstField}
		{...restProps}
	>
		<div class="flex h-14 shrink-0 items-center gap-3 ps-6 pe-4">
			<DialogPrimitive.Title class="text-title text-fg min-w-0 flex-1 truncate">
				{title}
			</DialogPrimitive.Title>
			<DialogPrimitive.Close>
				{#snippet child({ props })}
					<Button {...props} variant="outline" size="icon-xs">
						<XIcon />
						<span class="sr-only">{m.close()}</span>
					</Button>
				{/snippet}
			</DialogPrimitive.Close>
		</div>
		<div class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-6 pt-2 pb-6">
			{@render children()}
		</div>
		{#if footer}
			<div class="flex h-14 shrink-0 items-center justify-end gap-2 border-t px-6">
				{@render footer()}
			</div>
		{/if}
	</DialogPrimitive.Content>
</DialogPrimitive.Portal>

<script lang="ts">
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
	import type { ComponentProps } from 'svelte';
	import DropdownMenuPortal from './dropdown-menu-portal.svelte';
	import { getCloseMenu } from './dropdown-menu.svelte';

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

	const closeMenu = getCloseMenu();

	// A mouse opens the menu on pointerdown, and on phones the sheet then covers the trigger
	// bits-ui selects the item a press from the trigger is released on, so a release without a press inside the sheet is cancelled
	$effect(() => {
		const sheet = ref;
		if (!sheet) return;
		let pressed = false;
		const press = () => (pressed = true);
		const release = (e: PointerEvent) => {
			if (!pressed && window.matchMedia('(width < 40rem)').matches) e.preventDefault();
		};
		sheet.addEventListener('pointerdown', press, true);
		sheet.addEventListener('pointerup', release, true);
		return () => {
			sheet.removeEventListener('pointerdown', press, true);
			sheet.removeEventListener('pointerup', release, true);
		};
	});

	// Phones show the menu as a bottom sheet, which a downward swipe from its top closes like a native one
	// Svelte adds touchmove handlers as passive, so these listeners are added by hand to keep the page from scrolling along
	$effect(() => {
		const sheet = ref;
		if (!sheet) return;
		let swipe: { startY: number; y: number; time: number; velocity: number } | null = null;

		const start = (e: TouchEvent) => {
			const y = e.touches[0].clientY;
			const phone = window.matchMedia('(width < 40rem)').matches;
			swipe =
				phone && sheet.scrollTop <= 0 ? { startY: y, y, time: e.timeStamp, velocity: 0 } : null;
		};

		const move = (e: TouchEvent) => {
			if (!swipe) return;
			const y = e.touches[0].clientY;

			// A swipe that doesn't start downwards scrolls the menu instead
			if (y <= swipe.startY && swipe.y === swipe.startY) {
				swipe = null;
				return;
			}

			e.preventDefault();
			swipe.velocity = (y - swipe.y) / Math.max(1, e.timeStamp - swipe.time);
			swipe.y = y;
			swipe.time = e.timeStamp;
			sheet.style.transition = 'none';
			sheet.style.translate = `0 ${Math.max(0, y - swipe.startY)}px`;
		};

		// The sheet glides from where the finger left it, back up or out with the closing transition
		const end = () => {
			if (!swipe) return;
			const { startY, y, velocity } = swipe;
			swipe = null;
			sheet.style.removeProperty('transition');
			sheet.style.removeProperty('translate');
			if (y - startY > sheet.offsetHeight / 4 || velocity > 0.5) closeMenu();
		};

		sheet.addEventListener('touchstart', start, { passive: true });
		sheet.addEventListener('touchmove', move, { passive: false });
		sheet.addEventListener('touchend', end);
		sheet.addEventListener('touchcancel', end);
		return () => {
			sheet.removeEventListener('touchstart', start);
			sheet.removeEventListener('touchmove', move);
			sheet.removeEventListener('touchend', end);
			sheet.removeEventListener('touchcancel', end);
		};
	});
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

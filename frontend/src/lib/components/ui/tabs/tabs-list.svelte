<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const tabsListVariants = tv({
		base: 'group/tabs-list relative isolate flex w-fit items-center group-data-[orientation=vertical]/tabs:flex-col',
		variants: {
			variant: {
				default: 'gap-1',
				line: 'w-full gap-6 shadow-[inset_0_-1px_0_var(--border)]'
			}
		},
		defaultVariants: {
			variant: 'default'
		}
	});

	export type TabsListVariant = VariantProps<typeof tabsListVariants>['variant'];
</script>

<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import { Tabs as TabsPrimitive } from 'bits-ui';
	import { onMount } from 'svelte';

	let {
		ref = $bindable(null),
		variant = 'default',
		class: className,
		children,
		...restProps
	}: TabsPrimitive.ListProps & {
		variant?: TabsListVariant;
	} = $props();

	let indicator = $state<HTMLElement | null>(null);
	let indicatorVisible = $state(false);

	// Moves the indicator by FLIP so only transform animates, and never on first paint or on resize
	function updateIndicator(animate: boolean) {
		if (!ref || !indicator) return;
		const active = ref.querySelector<HTMLElement>(
			'[data-slot="tabs-trigger"][data-state="active"]'
		);
		if (!active || ref.dataset.orientation === 'vertical') {
			indicatorVisible = false;
			return;
		}

		const listRect = ref.getBoundingClientRect();
		const targetRect = active.getBoundingClientRect();
		const from = indicator.getBoundingClientRect();

		indicator.style.transition = 'none';
		indicator.style.transform = '';
		indicator.style.left = `${targetRect.left - listRect.left}px`;
		indicator.style.width = `${targetRect.width}px`;
		if (variant === 'default') {
			indicator.style.top = `${targetRect.top - listRect.top}px`;
			indicator.style.height = `${targetRect.height}px`;
		}

		const to = indicator.getBoundingClientRect();
		if (animate && indicatorVisible && from.width > 0 && to.width > 0) {
			indicator.style.transform = `translateX(${from.left - to.left}px) scaleX(${from.width / to.width})`;
			indicator.getBoundingClientRect();
			indicator.style.transition = '';
			indicator.style.transform = '';
		} else {
			indicator.style.transition = '';
		}
		indicatorVisible = true;
	}

	// Tab rows scroll sideways on phones, so the active tab is brought into view on load and whenever it changes
	function scrollActiveIntoView() {
		ref
			?.querySelector('[data-slot="tabs-trigger"][data-state="active"]')
			?.scrollIntoView({ block: 'nearest', inline: 'nearest' });
	}

	onMount(() => {
		if (!ref) return;

		const resizeObserver = new ResizeObserver(() => updateIndicator(false));
		const observeTriggers = () => {
			ref
				?.querySelectorAll<HTMLElement>('[data-slot="tabs-trigger"]')
				.forEach((trigger) => resizeObserver.observe(trigger));
		};
		const mutationObserver = new MutationObserver((mutations) => {
			observeTriggers();
			updateIndicator(true);
			const activated = mutations.some(
				(mutation) =>
					mutation.type === 'attributes' &&
					(mutation.target as HTMLElement).dataset.state === 'active'
			);
			if (activated) scrollActiveIntoView();
		});

		resizeObserver.observe(ref);
		observeTriggers();
		updateIndicator(false);
		scrollActiveIntoView();
		mutationObserver.observe(ref, {
			attributes: true,
			attributeFilter: ['data-state'],
			childList: true,
			subtree: true
		});

		return () => {
			mutationObserver.disconnect();
			resizeObserver.disconnect();
		};
	});
</script>

<TabsPrimitive.List
	bind:ref
	data-slot="tabs-list"
	data-variant={variant}
	class={cn(tabsListVariants({ variant }), className)}
	{...restProps}
>
	<span
		bind:this={indicator}
		aria-hidden="true"
		data-slot="tabs-indicator"
		class={cn(
			'pointer-events-none absolute left-0 origin-left opacity-0 transition-transform duration-(--dur-base) ease-in-out',
			variant === 'line'
				? 'bg-fg bottom-0 h-0.5'
				: 'bg-surface-raised top-0 -z-10 rounded-sm shadow-[inset_0_0_0_1px_var(--border-selected),var(--elevation-selected)]',
			indicatorVisible && 'opacity-100'
		)}
	></span>
	{@render children?.()}
</TabsPrimitive.List>

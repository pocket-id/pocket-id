<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const itemVariants = tv({
		base: 'group/item flex w-full flex-wrap items-center rounded-md border text-control transition-colors duration-(--dur-fast) outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [a]:hover:bg-fill-row-hover',
		variants: {
			variant: {
				default: 'border-transparent',
				outline: 'border-border bg-surface',
				card: 'border-border bg-surface',
				transparent: 'border-transparent bg-transparent'
			},
			size: {
				default: 'gap-4 px-4 py-3',
				sm: 'gap-3 px-3 py-2.5',
				xs: 'gap-2.5 px-3 py-2 in-data-[slot=dropdown-menu-content]:p-0'
			}
		},
		defaultVariants: {
			variant: 'default',
			size: 'default'
		}
	});

	export type ItemSize = VariantProps<typeof itemVariants>['size'];
	export type ItemVariant = VariantProps<typeof itemVariants>['variant'];
</script>

<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		child,
		variant,
		size,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		child?: Snippet<[{ props: Record<string, unknown> }]>;
		variant?: ItemVariant;
		size?: ItemSize;
	} = $props();

	const mergedProps = $derived({
		class: cn(itemVariants({ variant, size }), className),
		'data-slot': 'item',
		'data-variant': variant,
		'data-size': size,
		...restProps
	});
</script>

{#if child}
	{@render child({ props: mergedProps })}
{:else}
	<div bind:this={ref} {...mergedProps}>
		{@render mergedProps.children?.()}
	</div>
{/if}

<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';
	export const inputGroupAddonVariants = tv({
		base: "flex h-auto cursor-text items-center justify-center gap-2 text-control text-fg-muted select-none group-data-[disabled=true]/input-group:opacity-45 [&>svg:not([class*='size-'])]:size-4",
		variants: {
			align: {
				'inline-start': 'order-first pl-3 has-[>button]:pl-1',
				'inline-end': 'order-last pr-3.5 has-[>button]:pr-1',
				'block-start': 'order-first w-full justify-start px-3.5 pt-2.5',
				'block-end': 'order-last w-full justify-start px-3.5 pb-2.5'
			}
		},
		defaultVariants: {
			align: 'inline-start'
		}
	});

	export type InputGroupAddonAlign = VariantProps<typeof inputGroupAddonVariants>['align'];
</script>

<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		align = 'inline-start',
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		align?: InputGroupAddonAlign;
	} = $props();
</script>

<div
	bind:this={ref}
	role="group"
	data-slot="input-group-addon"
	data-align={align}
	class={cn(inputGroupAddonVariants({ align }), className)}
	onclick={(e) => {
		if ((e.target as HTMLElement).closest('button')) {
			return;
		}
		e.currentTarget.parentElement?.querySelector('input')?.focus();
	}}
	{...restProps}
>
	{@render children?.()}
</div>

<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	const inputGroupButtonVariants = tv({
		base: 'flex items-center gap-1.5',
		variants: {
			size: {
				xs: "h-(--h-xs) rounded-xs px-2 [&>svg:not([class*='size-'])]:size-3.5",
				sm: 'h-(--h-sm) px-2.5',
				'icon-xs': 'size-(--h-xs) rounded-xs p-0',
				'icon-sm': 'size-(--h-sm) p-0'
			}
		},
		defaultVariants: {
			size: 'xs'
		}
	});

	export type InputGroupButtonSize = VariantProps<typeof inputGroupButtonVariants>['size'];
</script>

<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import type { ComponentProps } from 'svelte';
	import { Button } from '#lib/components/ui/button/index.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		type = 'button',
		variant = 'ghost',
		size = 'xs',
		...restProps
	}: Omit<ComponentProps<typeof Button>, 'href' | 'size'> & {
		size?: InputGroupButtonSize;
	} = $props();
</script>

<Button
	bind:ref
	{type}
	data-size={size}
	{variant}
	class={cn(inputGroupButtonVariants({ size }), className)}
	{...restProps}
>
	{@render children?.()}
</Button>

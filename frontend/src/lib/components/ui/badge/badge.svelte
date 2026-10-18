<script lang="ts" module>
	import { type VariantProps, tv } from 'tailwind-variants';

	const tag =
		'h-[22px] rounded-xs border border-border px-2 text-micro [&>svg]:size-3! [a]:hover:bg-fill-hover';
	const status =
		"gap-1.5 text-control text-fg before:size-2 before:shrink-0 before:rounded-full before:content-['']";

	export const badgeVariants = tv({
		base: 'group/badge inline-flex w-fit shrink-0 items-center gap-1 overflow-hidden whitespace-nowrap align-middle transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [&>svg]:pointer-events-none',
		variants: {
			variant: {
				default: `${tag} bg-fill-muted text-fg`,
				secondary: `${tag} border-transparent bg-fill-muted text-fg`,
				outline: `${tag} bg-transparent text-fg`,
				mono: `${tag} h-5 font-mono text-mono-s`,
				success: `${status} before:bg-success-solid`,
				warning: `${status} before:bg-warning-solid`,
				destructive: `${status} before:bg-danger-solid`,
				muted: `${status} before:bg-fg-muted`
			}
		},
		defaultVariants: {
			variant: 'default'
		}
	});

	export type BadgeVariant = VariantProps<typeof badgeVariants>['variant'];
</script>

<script lang="ts">
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import { cn, type WithElementRef } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		href,
		class: className,
		variant = 'default',
		children,
		...restProps
	}: WithElementRef<HTMLAnchorAttributes> & {
		variant?: BadgeVariant;
	} = $props();
</script>

<svelte:element
	this={href ? 'a' : 'span'}
	bind:this={ref}
	data-slot="badge"
	{href}
	class={cn(badgeVariants({ variant }), className)}
	{...restProps}
>
	{@render children?.()}
</svelte:element>

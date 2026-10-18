<script lang="ts" module>
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';
	import { type VariantProps, tv } from 'tailwind-variants';

	const accentWash = 'bg-primary/12 text-fg before:bg-primary/6';

	export const buttonVariants = tv({
		base: "group/button relative isolate inline-flex shrink-0 items-center justify-center gap-2 rounded-sm text-control font-medium whitespace-nowrap select-none transition-motion before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-fast) before:ease-out hover:not-disabled:before:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring active:not-disabled:scale-(--press-scale) disabled:cursor-not-allowed disabled:opacity-45 aria-disabled:pointer-events-none aria-disabled:opacity-45 aria-busy:cursor-progress aria-busy:opacity-100 aria-expanded:before:opacity-100 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
		variants: {
			variant: {
				default:
					'bg-primary text-primary-foreground shadow-[inset_0_0_0_1px_var(--primary-edge)] before:bg-primary-hover',
				secondary: accentWash,
				outline: accentWash,
				ghost: 'text-fg before:bg-fill-hover',
				destructive: 'bg-danger-fill text-danger-fill-fg before:bg-danger-fill-hover',
				inverse:
					'bg-inverse-fg text-inverse before:bg-[color-mix(in_srgb,var(--inverse-fg)_88%,var(--inverse))] focus-visible:outline-inverse-fg',
				link: 'text-fg before:hidden hover:text-fg-muted active:not-disabled:scale-100'
			},
			size: {
				default: 'h-(--h-md) px-4',
				sm: "h-(--h-sm) px-3 [&_svg:not([class*='size-'])]:size-3.5",
				icon: 'size-(--h-md)',
				'icon-sm': 'size-(--h-sm)',
				'icon-xs': "size-(--h-xs) [&_svg:not([class*='size-'])]:size-3.5"
			}
		},
		compoundVariants: [
			{ variant: 'link', class: 'h-auto px-0' },
			{
				variant: 'ghost',
				size: ['icon', 'icon-sm', 'icon-xs'],
				class:
					'text-fg-muted hover:text-fg aria-expanded:text-fg aria-expanded:before:bg-fill-selected'
			}
		],
		defaultVariants: {
			variant: 'default',
			size: 'default'
		}
	});

	export type ButtonVariant = VariantProps<typeof buttonVariants>['variant'];
	export type ButtonSize = VariantProps<typeof buttonVariants>['size'];

	export type ButtonProps = WithElementRef<HTMLButtonAttributes> &
		WithElementRef<HTMLAnchorAttributes> & {
			variant?: ButtonVariant;
			size?: ButtonSize;
			isLoading?: boolean;
			autofocus?: boolean;
		};
</script>

<script lang="ts">
	import { Spinner } from '#lib/components/ui/spinner/index.ts';
	import { onMount } from 'svelte';

	let {
		class: className,
		variant = 'default',
		size = 'default',
		ref = $bindable(null),
		href = undefined,
		type = 'button',
		disabled,
		isLoading = false,
		autofocus = false,
		onclick,
		usePromiseLoading = false,
		children,
		...restProps
	}: ButtonProps & {
		usePromiseLoading?: boolean;
	} = $props();

	onMount(async () => {
		// Using autofocus can be bad for a11y, but in the case of Pocket ID is only used responsibly in places where there are not many choices, and on buttons only where there's descriptive text
		if (autofocus) {
			// Use setTimeout to make sure the element is showing
			setTimeout(() => ref?.focus(), 100);
		}
	});

	async function handleOnClick(event: any) {
		if (usePromiseLoading && onclick) {
			isLoading = true;
			try {
				await onclick(event);
			} finally {
				isLoading = false;
			}
		} else {
			onclick?.(event);
		}
	}
</script>

{#if href}
	<a
		bind:this={ref}
		data-slot="button"
		class={cn(buttonVariants({ variant, size }), className)}
		href={disabled ? undefined : href}
		aria-disabled={disabled}
		role={disabled ? 'link' : undefined}
		tabindex={disabled ? -1 : undefined}
		{...restProps}
	>
		{@render children?.()}
	</a>
{:else}
	<button
		bind:this={ref}
		data-slot="button"
		class={cn(buttonVariants({ variant, size }), className)}
		{type}
		disabled={disabled || isLoading}
		aria-busy={isLoading || undefined}
		onclick={handleOnClick}
		{...restProps}
	>
		<!-- Label and spinner share one grid cell, so the width never changes while busy -->
		<!-- Both spans may shrink below their content, so a truncating label can end at the button's edge -->
		<span class="inline-grid max-w-full min-w-0 place-items-center">
			<span
				class={cn(
					'col-start-1 row-start-1 inline-flex max-w-full min-w-0 items-center justify-center gap-2 transition-opacity duration-(--dur-fast) ease-out',
					isLoading && 'opacity-0'
				)}
			>
				{@render children?.()}
			</span>
			<Spinner
				class={cn(
					'col-start-1 row-start-1 size-3.5 transition-opacity duration-(--dur-fast) ease-out',
					isLoading ? 'delay-60' : 'opacity-0 [animation-play-state:paused]'
				)}
			/>
		</span>
	</button>
{/if}

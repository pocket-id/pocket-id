<script lang="ts" module>
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';
	import { tv, type VariantProps } from 'tailwind-variants';

	export const buttonVariants = tv({
		base: "focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-medium outline-none transition-all focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 [&_svg:not([class*='size-'])]:size-4 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg]:shrink-0 transition active:scale-95",
		variants: {
			variant: {
				default: 'bg-primary text-primary-foreground hover:bg-primary/80',
				outline:
					'border border-border bg-background hover:bg-muted hover:text-foreground dark:hover:bg-input/30 aria-expanded:bg-muted aria-expanded:text-foreground dark:bg-transparent',
				secondary:
					'bg-secondary text-secondary-foreground hover:bg-secondary/80 aria-expanded:bg-secondary aria-expanded:text-secondary-foreground',
				ghost:
					'hover:bg-muted hover:text-foreground dark:hover:bg-muted/50 aria-expanded:bg-muted aria-expanded:text-foreground',
				destructive:
					'bg-destructive/10 hover:bg-destructive/20 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/20 text-destructive focus-visible:border-destructive/40 dark:hover:bg-destructive/30',
				link: 'text-primary underline-offset-4 hover:underline'
			},
			size: {
				default: 'h-10 px-4 py-2',
				sm: 'h-9 px-3 py-2',
				lg: 'h-11 px-8',
				icon: 'h-10 w-10'
			}
		},
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
	import { Spinner } from '$lib/components/ui/spinner';
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
		onclick={handleOnClick}
		{...restProps}
	>
		<span class="flex items-center w-full justify-center">
			<Spinner
				class={cn(
					'grid overflow-hidden transition-[width,opacity,margin-right] duration-400 ease-[linear(0,0.897_14.4%,1.311_31.2%,1.338_46%,1.054_80.4%,1)]',
					isLoading ? 'w-4 opacity-100 mr-2' : 'w-0 opacity-0'
				)}
				aria-hidden={!isLoading}
			/>
			{@render children?.()}
		</span>
	</button>
{/if}

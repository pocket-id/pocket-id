<script lang="ts" module>
	import { type VariantProps, tv } from 'tailwind-variants';

	export const alertVariants = tv({
		base: 'relative grid w-full grid-cols-[0_1fr] items-start gap-y-0.5 rounded-lg border px-4 py-3 text-sm has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr] has-[>svg]:gap-x-3 [&>svg]:size-4 [&>svg]:translate-y-0.5 [&>svg]:text-current',
		variants: {
			variant: {
				default: 'bg-card text-card-foreground',
				destructive:
					'text-destructive bg-card *:data-[slot=alert-description]:text-destructive/90 [&>svg]:text-current',
				warning:
					'bg-amber-100 text-amber-900 dark:bg-amber-900 dark:text-amber-100 [&>svg]:text-amber-900 dark:[&>svg]:text-amber-100'
			}
		},
		defaultVariants: {
			variant: 'default'
		}
	});

	export type AlertVariant = VariantProps<typeof alertVariants>['variant'];
</script>

<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import { onMount } from 'svelte';
	import { LucideX } from '@lucide/svelte';

	let {
		ref = $bindable(null),
		class: className,
		variant = 'default',
		children,
		dismissibleId = undefined,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		variant?: AlertVariant;
		dismissibleId?: string;
	} = $props();

	let isVisible = $state(!dismissibleId);

	onMount(() => {
		if (dismissibleId) {
			const dismissedAlerts = JSON.parse(localStorage.getItem('dismissed-alerts') || '[]');
			isVisible = !dismissedAlerts.includes(dismissibleId);
		}
	});

	function dismiss() {
		if (dismissibleId) {
			const dismissedAlerts = JSON.parse(localStorage.getItem('dismissed-alerts') || '[]');
			localStorage.setItem('dismissed-alerts', JSON.stringify([...dismissedAlerts, dismissibleId]));
			isVisible = false;
		}
	}
</script>

{#if isVisible}
	<div
		bind:this={ref}
		data-slot="alert"
		class={cn(alertVariants({ variant }), className)}
		{...restProps}
		role="alert"
	>
		{@render children?.()}
		{#if dismissibleId}
			<button onclick={dismiss} class="absolute top-0 right-0 m-3 text-black dark:text-white"
				><LucideX class="size-4" /></button
			>
		{/if}
	</div>
{/if}

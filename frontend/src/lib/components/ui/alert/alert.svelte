<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const alertVariants = tv({
		base: 'relative grid w-full grid-cols-[0_1fr] items-start gap-y-0.5 rounded-lg border px-4 py-3 text-sm has-[>svg]:grid-cols-[calc(var(--spacing)*4)_1fr] has-[>svg]:gap-x-3 [&>svg]:size-4 [&>svg]:translate-y-0.5 [&>svg]:text-current',
		variants: {
			variant: {
				default: 'bg-card text-card-foreground',
				success:
					'bg-green-100 text-green-900 dark:bg-green-900 dark:text-green-100 [&>svg]:text-green-900 dark:[&>svg]:text-green-100',
				info: 'bg-blue-100 text-blue-900 dark:bg-blue-900 dark:text-blue-100 [&>svg]:text-blue-900 dark:[&>svg]:text-blue-100',
				destructive:
					'bg-red-100 text-red-900 dark:bg-red-900 dark:text-red-100 [&>svg]:text-red-900 dark:[&>svg]:text-red-100',
				warning:
					'bg-warning text-warning-foreground border-warning/40 [&>svg]:text-warning-foreground'
			}
		},
		defaultVariants: {
			variant: 'default'
		}
	});

	export type AlertVariant = VariantProps<typeof alertVariants>['variant'];
</script>

<script lang="ts">
	import { cn, type WithElementRef } from '$lib/utils/style.js';
	import { LucideX } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		variant = 'default',
		children,
		onDismiss,
		dismissibleId = undefined,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		variant?: AlertVariant;
		onDismiss?: () => void;
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
		onDismiss?.();
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
		{#if dismissibleId || onDismiss}
			<button onclick={dismiss} class="absolute right-0 top-0 m-3 text-black dark:text-white"
				><LucideX class="size-4" /></button
			>
		{/if}
	</div>
{/if}

<script lang="ts" module>
	import { type VariantProps, tv } from 'tailwind-variants';

	export const alertVariants = tv({
		base: "grid gap-0.5 rounded-2xl border px-4 py-3 text-left text-sm has-data-[slot=alert-action]:relative has-data-[slot=alert-action]:pr-18 has-[>svg]:grid-cols-[auto_1fr] has-[>svg]:gap-x-2.5 *:[svg]:row-span-2 *:[svg]:translate-y-0.5 *:[svg]:text-current *:[svg:not([class*='size-'])]:size-4 group/alert relative w-full",
		variants: {
			variant: {
				default: 'bg-card text-card-foreground',
				destructive:
					'text-destructive bg-card *:data-[slot=alert-description]:text-destructive/90 *:[svg]:text-current',
				success:
					'bg-green-100 text-green-900 dark:bg-green-900 dark:text-green-100 *:[svg]:text-current',
				info: 'bg-blue-100 text-blue-900 dark:bg-blue-900 dark:text-blue-100 *:[svg]:text-current',
				warning: 'bg-warning text-warning-foreground border-warning/40 *:[svg]:text-current'
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
	import { LucideX } from '@lucide/svelte';
	import { onMount } from 'svelte';

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
		role="alert"
		class={cn(alertVariants({ variant }), className)}
		{...restProps}
	>
		{@render children?.()}
		{#if dismissibleId || onDismiss}
			<button onclick={dismiss} class="absolute top-2.5 right-3 text-current">
				<LucideX class="size-4" />
			</button>
		{/if}
	</div>
{/if}

<script lang="ts" module>
	import { tv, type VariantProps } from 'tailwind-variants';

	export const alertVariants = tv({
		base: 'group/alert relative grid w-full gap-0.5 rounded-sm border border-(--tone-border) bg-(--tone-soft) px-3 py-2.5 text-left text-control text-fg has-data-[slot=alert-action]:pr-18 has-[>svg]:grid-cols-[auto_1fr] has-[>svg]:gap-x-2 *:[svg]:row-span-2 *:[svg]:size-3.5 *:[svg]:translate-y-0.75 *:[svg]:text-(--tone-solid)',
		variants: {
			variant: {
				info: '[--tone-border:var(--info-border)] [--tone-fg:var(--info-fg)] [--tone-soft:var(--info-soft)] [--tone-solid:var(--info-solid)]',
				success:
					'[--tone-border:var(--success-border)] [--tone-fg:var(--success-fg)] [--tone-soft:var(--success-soft)] [--tone-solid:var(--success-solid)]',
				warning:
					'[--tone-border:var(--warning-border)] [--tone-fg:var(--warning-fg)] [--tone-soft:var(--warning-soft)] [--tone-solid:var(--warning-solid)]',
				destructive:
					'[--tone-border:var(--danger-border)] [--tone-fg:var(--danger-fg)] [--tone-soft:var(--danger-soft)] [--tone-solid:var(--danger-solid)]'
			}
		},
		defaultVariants: {
			variant: 'info'
		}
	});

	export type AlertVariant = VariantProps<typeof alertVariants>['variant'];
</script>

<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideX } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		variant = 'info',
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
			const dismissedAlerts = JSON.parse(localStorage?.getItem('dismissed-alerts') || '[]');
			isVisible = !dismissedAlerts.includes(dismissibleId);
		}
	});

	function dismiss() {
		onDismiss?.();
		if (dismissibleId) {
			const dismissedAlerts = JSON.parse(localStorage?.getItem('dismissed-alerts') || '[]');
			localStorage?.setItem(
				'dismissed-alerts',
				JSON.stringify([...dismissedAlerts, dismissibleId])
			);
			isVisible = false;
		}
	}
</script>

{#if isVisible}
	<div
		bind:this={ref}
		data-slot="alert"
		role={variant === 'destructive' ? 'alert' : 'status'}
		class={cn(alertVariants({ variant }), (dismissibleId || onDismiss) && 'pr-10', className)}
		{...restProps}
	>
		{@render children?.()}
		{#if dismissibleId || onDismiss}
			<button
				onclick={dismiss}
				aria-label={m.close()}
				class="text-fg-muted hover:bg-fill-hover hover:text-fg focus-visible:outline-ring absolute top-2 right-2 grid size-6 place-items-center rounded-xs transition-colors focus-visible:outline-2"
			>
				<LucideX class="size-3.5" />
			</button>
		{/if}
	</div>
{/if}

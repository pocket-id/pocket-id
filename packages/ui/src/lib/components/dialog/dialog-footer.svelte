<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { Button } from '#lib/components/button/index.js';
	import { getUiLabels } from '#lib/labels.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		showCloseButton = false,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> & {
		showCloseButton?: boolean;
	} = $props();

	const labels = getUiLabels();
</script>

<div
	bind:this={ref}
	data-slot="dialog-footer"
	class={cn('mt-1 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end', className)}
	{...restProps}
>
	{@render children?.()}
	{#if showCloseButton}
		<DialogPrimitive.Close>
			{#snippet child({ props })}
				<Button variant="outline" {...props}>{labels.close()}</Button>
			{/snippet}
		</DialogPrimitive.Close>
	{/if}
</div>

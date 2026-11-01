<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { cn, type WithElementRef } from '#lib/utils/style.js';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="alert-description"
	class={cn(
		'text-fg [&_a]:inline-link text-pretty group-has-[>svg]/alert:col-start-2 [&_p:not(:last-child)]:mb-2',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

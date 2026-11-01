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
	data-slot="alert-title"
	class={cn(
		'[&_a]:inline-link font-medium text-(--tone-fg) group-has-[>svg]/alert:col-start-2',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

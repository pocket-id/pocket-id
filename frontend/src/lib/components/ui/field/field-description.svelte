<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLParagraphElement>> = $props();
</script>

<p
	bind:this={ref}
	data-slot="field-description"
	class={cn(
		'text-caption text-fg-muted text-left [[data-variant=legend]+&]:-mt-1.5',
		'[&>a]:inline-link',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</p>

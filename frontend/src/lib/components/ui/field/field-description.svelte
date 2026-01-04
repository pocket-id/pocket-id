<script lang="ts">
	import { cn, type WithElementRef } from '$lib/utils/style.js';
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
		'text-muted-foreground -mt-1 mb-0 text-xs leading-normal font-normal group-has-[[data-orientation=horizontal]]/field:text-balance',
		'[&>a:hover]:text-primary [&>a]:underline [&>a]:underline-offset-4',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</p>

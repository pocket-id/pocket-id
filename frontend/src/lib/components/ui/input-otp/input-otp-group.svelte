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
	data-slot="input-otp-group"
	class={cn(
		'bg-surface relative flex min-w-0 flex-1 overflow-hidden rounded-sm after:pointer-events-none after:absolute after:inset-0 after:z-1 after:rounded-[inherit] after:shadow-[inset_0_0_0_1px_var(--border-control)]',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>

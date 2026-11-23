<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils/style.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...props
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="input-group"
	role="group"
	class={cn(
		'group/input-group bg-fill-muted text-fg ring-ring-halo hover:bg-fill-hover has-[[data-slot=input-group-control]:focus-visible]:border-ring has-[[data-slot=input-group-control]:focus-visible]:bg-surface has-[[data-slot][aria-invalid=true]]:border-danger-solid has-[[data-slot][aria-invalid=true]]:ring-danger-solid/20 dark:border-fg/10 dark:bg-fg/6 dark:hover:bg-fg/9 dark:has-[[data-slot=input-group-control]:focus-visible]:border-ring dark:has-[[data-slot=input-group-control]:focus-visible]:bg-fg/7 dark:has-[[data-slot][aria-invalid=true]]:border-danger-solid relative flex h-(--h-md) w-full min-w-0 items-center rounded-sm border border-transparent has-[[data-slot=input-group-control]:focus-visible]:ring-(length:--halo-width) has-[>[data-align=block-end]]:h-auto has-[>[data-align=block-end]]:flex-col has-[>[data-align=block-start]]:h-auto has-[>[data-align=block-start]]:flex-col has-[>textarea]:h-auto has-[>[data-align=inline-end]]:[&>input]:pr-2.5 has-[>[data-align=inline-start]]:[&>input]:pl-2.5',
		className
	)}
	{...props}
>
	{@render children?.()}
</div>

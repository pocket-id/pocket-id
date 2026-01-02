<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils/style.js";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	role="heading"
	aria-level="3"
	data-slot="item-title"
	class={cn("flex w-fit items-center gap-2 font-semibold leading-snug", className)}
	{...restProps}
>
	{@render children?.()}
</div>

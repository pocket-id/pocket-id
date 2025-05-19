<script lang="ts">
	import type { HTMLAttributes } from "svelte/elements";
	import { cn, type WithElementRef } from "$lib/utils/style.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	class={cn("mt-4 flex flex-col space-y-4 sm:flex-row sm:space-x-4 sm:space-y-0", className)}
	{...restProps}
>
	{@render children?.()}
</div>

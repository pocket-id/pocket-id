<script lang="ts">
	import type { ClassValue, HTMLAttributes } from "svelte/elements";
	import { cn } from "$lib/utils/style.js";

	type $$Props = HTMLAttributes<HTMLSpanElement>;

	let className: ClassValue | undefined | null = undefined;
	export { className as class };
</script>

<span
	class={cn("text-muted-foreground ml-auto text-xs tracking-widest", className)}
	{...$$restProps}
>
	<slot />
</span>

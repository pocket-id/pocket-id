<script lang="ts">
	import { Command as CommandPrimitive } from "cmdk-sv";
	import { cn } from "$lib/utils/style.js";
	import type { ClassValue } from "svelte/elements";

	type $$Props = CommandPrimitive.ListProps;
	let className: ClassValue | undefined | null = undefined;
	export { className as class };
</script>

<CommandPrimitive.List
	class={cn("max-h-[300px] overflow-y-auto overflow-x-hidden", className)}
	{...$$restProps}
>
	<slot />
</CommandPrimitive.List>

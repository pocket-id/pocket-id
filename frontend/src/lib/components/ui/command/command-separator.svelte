<script lang="ts">
	import { Command as CommandPrimitive } from "cmdk-sv";
	import { cn } from "$lib/utils/style.js";
	import type { ClassValue } from "svelte/elements";

	type $$Props = CommandPrimitive.SeparatorProps;
	let className: ClassValue | undefined | null = undefined;
	export { className as class };
</script>

<CommandPrimitive.Separator class={cn("bg-border -mx-1 h-px", className)} {...$$restProps} />

<script lang="ts">
	import { Command as CommandPrimitive } from "cmdk-sv";
	import { cn } from "$lib/utils/style.js";
	import type { ClassValue } from "svelte/elements";

	type $$Props = CommandPrimitive.EmptyProps;
	let className: ClassValue | undefined | null = undefined;
	export { className as class };
</script>

<CommandPrimitive.Empty class={cn("py-6 text-center text-sm", className)} {...$$restProps}>
	<slot />
</CommandPrimitive.Empty>

<script lang="ts">
	import { Calendar as CalendarPrimitive } from "bits-ui";
	import { cn } from "$lib/utils/style.js";

	type $$Props = CalendarPrimitive.CellProps;

	export let date: $$Props["date"];
	let className: $$Props["class"] = undefined;
	export { className as class };
</script>

<CalendarPrimitive.Cell
	{date}
	class={cn(
		"[&:has([data-selected])]:bg-accent [&:has([data-selected][data-outside-month])]:bg-accent/50 relative h-9 w-9 p-0 text-center text-sm focus-within:relative focus-within:z-20 [&:has([data-selected])]:rounded-md",
		className
	)}
	{...$$restProps}
>
	<slot />
</CalendarPrimitive.Cell>

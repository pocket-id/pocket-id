<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui-old';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.CellProps;

	interface Props {
		date: $$Props['date'];
		class?: $$Props['class'];
		children?: import('svelte').Snippet;
		[key: string]: any;
	}

	let { date, class: className = undefined, children, ...rest }: Props = $props();
</script>

<CalendarPrimitive.Cell
	{date}
	class={cn(
		'[&:has([data-selected])]:bg-accent [&:has([data-selected][data-outside-month])]:bg-accent/50 relative h-9 w-9 p-0 text-center text-sm focus-within:relative focus-within:z-20 [&:has([data-selected])]:rounded-md',
		className
	)}
	{...rest}
>
	{@render children?.()}
</CalendarPrimitive.Cell>

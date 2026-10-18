<script lang="ts">
	import { cn } from '#lib/utils/style.js';
	import { Calendar as CalendarPrimitive } from 'bits-ui';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: CalendarPrimitive.DayProps = $props();
</script>

<CalendarPrimitive.Day
	bind:ref
	class={cn(
		'flex size-(--cell-size) flex-col items-center justify-center gap-1 rounded-(--cell-radius) p-0 leading-none font-normal whitespace-nowrap select-none',
		'[&:last-child[data-selected=true]_button]:rounded-r-(--cell-radius)',
		'not-data-selected:hover:bg-fill-hover transition-colors duration-(--dur-fast)',
		'[&[data-today]:not([data-selected])]:bg-fill-muted [&[data-today][data-disabled]]:text-fg-muted [&[data-today]:not([data-selected])]:font-medium',
		'data-[selected]:bg-primary data-[selected]:text-primary-foreground',
		// Outside months
		'[&[data-outside-month]:not([data-selected],[data-disabled],[data-unavailable])]:text-fg-muted',
		// Disabled
		'data-[disabled]:text-fg-subtle data-[disabled]:pointer-events-none',
		// Unavailable
		'data-[unavailable]:text-fg-subtle data-[unavailable]:line-through',
		// focus
		'focus-visible:outline-ring focus-visible:relative focus-visible:outline-2 focus-visible:outline-offset-2',
		// inner spans
		'[&>span]:text-micro [&>span]:opacity-70',
		className
	)}
	{...restProps}
/>

<script lang="ts">
	import { buttonVariants } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils/style.js';
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
		buttonVariants({ variant: 'ghost' }),
		'size-8 p-0 font-normal select-none',
		'[&[data-today]:not([data-selected])]:bg-accent [&[data-today]:not([data-selected])]:text-accent-foreground',
		// Selected
		'data-selected:bg-primary data-selected:text-primary-foreground data-selected:hover:bg-primary data-selected:hover:text-primary-foreground data-selected:focus:bg-primary data-selected:focus:text-primary-foreground dark:data-selected:hover:bg-primary dark:data-selected:focus:bg-primary data-selected:opacity-100',
		// Disabled
		'data-disabled:text-muted-foreground data-disabled:opacity-50',
		// Unavailable
		'data-unavailable:text-destructive-foreground data-unavailable:line-through',
		// Outside months
		'data-[outside-month]:text-muted-foreground [&[data-outside-month][data-selected]]:bg-accent/50 [&[data-outside-month][data-selected]]:text-muted-foreground data-[outside-month]:pointer-events-none data-[outside-month]:opacity-50 [&[data-outside-month][data-selected]]:opacity-30',
		className
	)}
	{...restProps}
/>

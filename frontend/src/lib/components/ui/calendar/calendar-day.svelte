<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui';
	import { buttonVariants } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils/style.js';

	type $$Props = CalendarPrimitive.DayProps;
	type $$Events = CalendarPrimitive.DayEvents;

	interface Props {
		date: $$Props['date'];
		month: $$Props['month'];
		class?: $$Props['class'];
		children?: import('svelte').Snippet<[any]>;
		[key: string]: any
	}

	let {
		date,
		month,
		class: className = undefined,
		children,
		...rest
	}: Props = $props();
	

	const children_render = $derived(children);
</script>

<CalendarPrimitive.Day
	on:click
	{date}
	{month}
	class={cn(
		buttonVariants({ variant: 'ghost' }),
		'h-9 w-9 p-0 font-normal ',
		'[&[data-today]:not([data-selected])]:bg-accent [&[data-today]:not([data-selected])]:text-accent-foreground',
		// Selected
		'data-selected:bg-primary data-selected:text-primary-foreground data-selected:hover:bg-primary data-selected:hover:text-primary-foreground data-selected:focus:bg-primary data-selected:focus:text-primary-foreground data-selected:opacity-100',
		// Disabled
		'data-disabled:text-muted-foreground data-disabled:opacity-50',
		// Unavailable
		'data-unavailable:text-destructive-foreground data-unavailable:line-through',
		// Outside months
		'data-outside-month:text-muted-foreground [&[data-outside-month][data-selected]]:bg-accent/50 [&[data-outside-month][data-selected]]:text-muted-foreground data-outside-month:pointer-events-none data-outside-month:opacity-50 [&[data-outside-month][data-selected]]:opacity-30',
		className
	)}
	{...rest}
	
	
	
	
>
	{#snippet children({ selected, disabled, unavailable, builder })}
		{#if children_render}{@render children_render({ selected, disabled, unavailable, builder, })}{:else}
			{date.day}
		{/if}
	{/snippet}
</CalendarPrimitive.Day>

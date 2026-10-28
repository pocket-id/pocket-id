<script lang="ts">
	import { Calendar as CalendarPrimitive } from 'bits-ui';
	import { cn, type WithoutChildrenOrChild } from '#lib/utils/style.js';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';

	let {
		ref = $bindable(null),
		class: className,
		value,
		onchange,
		...restProps
	}: WithoutChildrenOrChild<CalendarPrimitive.MonthSelectProps> = $props();
</script>

<span
	class={cn(
		'bg-fill-muted has-focus-visible:border-ring has-focus-visible:ring-ring-halo relative flex rounded-sm border border-transparent has-focus-visible:ring-3',
		className
	)}
>
	<CalendarPrimitive.MonthSelect
		bind:ref
		class="bg-background dark:bg-popover dark:text-popover-foreground absolute inset-0 opacity-0"
		{...restProps}
	>
		{#snippet child({ props, monthItems, selectedMonthItem })}
			<select {...props} {value} {onchange}>
				{#each monthItems as monthItem (monthItem.value)}
					<option
						value={monthItem.value}
						selected={value !== undefined
							? monthItem.value === value
							: monthItem.value === selectedMonthItem.value}
					>
						{monthItem.label}
					</option>
				{/each}
			</select>
			<span
				class="text-control [&>svg]:text-fg-muted flex h-(--cell-size) items-center gap-1 rounded-sm ps-2 pe-1 font-medium select-none [&>svg]:size-3.5"
				aria-hidden="true"
			>
				{monthItems.find((item) => item.value === value)?.label || selectedMonthItem.label}
				<ChevronDownIcon class={cn('size-4', className)} />
			</span>
		{/snippet}
	</CalendarPrimitive.MonthSelect>
</span>

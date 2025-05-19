<script lang="ts">
	import * as Calendar from '$lib/components/ui/calendar/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { cn } from '$lib/utils/style';
	import { DateFormatter, getLocalTimeZone, today } from '@internationalized/date';
	import { Calendar as CalendarPrimitive } from 'bits-ui-old';

	type $$Props = CalendarPrimitive.Props;
	type $$Events = CalendarPrimitive.Events;

	const monthOptions = [
		'January',
		'February',
		'March',
		'April',
		'May',
		'June',
		'July',
		'August',
		'September',
		'October',
		'November',
		'December'
	].map((month, i) => ({ value: i + 1, label: month }));

	const monthFmt = new DateFormatter('en-US', {
		month: 'long'
	});

	const yearOptions = Array.from({ length: 100 }, (_, i) => ({
		label: String(new Date().getFullYear() + i),
		value: new Date().getFullYear() + i
	}));

	interface Props {
		value?: $$Props['value'];
		placeholder?: $$Props['placeholder'];
		weekdayFormat?: $$Props['weekdayFormat'];
		class?: $$Props['class'];
		[key: string]: any;
	}

	let {
		value = $bindable(undefined),
		placeholder = $bindable(today(getLocalTimeZone())),
		weekdayFormat = 'short',
		class: className = undefined,
		...rest
	}: Props = $props();

	let defaultYear = $derived(
		placeholder
			? {
					value: placeholder.year,
					label: String(placeholder.year)
				}
			: undefined
	);
	let defaultMonth = $derived(
		placeholder
			? {
					value: placeholder.month,
					label: monthFmt.format(placeholder.toDate(getLocalTimeZone()))
				}
			: undefined
	);
</script>

<CalendarPrimitive.Root
	{weekdayFormat}
	class={cn('rounded-md border p-3', className)}
	{...rest}
	on:keydown
	bind:value
	bind:placeholder
>
	{#snippet children({ months, weekdays })}
		<Calendar.Header>
			<Calendar.Heading class="flex w-full items-center justify-between gap-2">
				<Select.Root
					selected={defaultMonth}
					items={monthOptions}
					onSelectedChange={(v) => {
						if (!v || !placeholder) return;
						if (v.value === placeholder?.month) return;
						placeholder = placeholder.set({ month: v.value });
					}}
				>
					<Select.Trigger aria-label="Select month" class="w-[60%]">
						<Select.Value placeholder="Select month" />
					</Select.Trigger>
					<Select.Content class="max-h-[200px] overflow-y-auto">
						{#each monthOptions as { value, label }}
							<Select.Item {value} {label}>
								{label}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
				<Select.Root
					selected={defaultYear}
					items={yearOptions}
					onSelectedChange={(v) => {
						if (!v || !placeholder) return;
						if (v.value === placeholder?.year) return;
						placeholder = placeholder.set({ year: v.value });
					}}
				>
					<Select.Trigger aria-label="Select year" class="w-[40%]">
						<Select.Value placeholder="Select year" />
					</Select.Trigger>
					<Select.Content class="max-h-[200px] overflow-y-auto">
						{#each yearOptions as { value, label }}
							<Select.Item {value} {label}>
								{label}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</Calendar.Heading>
		</Calendar.Header>
		<Calendar.Months>
			{#each months as month}
				<Calendar.Grid>
					<Calendar.GridHead>
						<Calendar.GridRow class="flex">
							{#each weekdays as weekday}
								<Calendar.HeadCell>
									{weekday.slice(0, 2)}
								</Calendar.HeadCell>
							{/each}
						</Calendar.GridRow>
					</Calendar.GridHead>
					<Calendar.GridBody>
						{#each month.weeks as weekDates}
							<Calendar.GridRow class="mt-2 w-full">
								{#each weekDates as date}
									<Calendar.Cell {date}>
										<Calendar.Day {date} month={month.value} />
									</Calendar.Cell>
								{/each}
							</Calendar.GridRow>
						{/each}
					</Calendar.GridBody>
				</Calendar.Grid>
			{/each}
		</Calendar.Months>
	{/snippet}
</CalendarPrimitive.Root>

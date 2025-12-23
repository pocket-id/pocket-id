<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Calendar } from '$lib/components/ui/calendar';
	import * as Popover from '$lib/components/ui/popover';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';
	import { cn } from '$lib/utils/style';
	import {
		CalendarDate,
		DateFormatter,
		getLocalTimeZone,
		type DateValue
	} from '@internationalized/date';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = {
		value?: Date;
		id?: string;
	} & HTMLAttributes<HTMLDivElement>;

	let { value = $bindable(undefined), id, ...restProps }: Props = $props();

	let calendarDisplayDate: CalendarDate | undefined = $state(
		value ? dateToCalendarDate(value) : undefined
	);

	let open = $state(false);

	function dateToCalendarDate(d: Date): CalendarDate {
		return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
	}

	$effect(() => {
		if (calendarDisplayDate) {
			const newExternalDate = calendarDisplayDate.toDate(getLocalTimeZone());
			if (!value || value.getTime() !== newExternalDate.getTime()) {
				value = newExternalDate;
			}
		} else {
			if (value !== undefined) {
				value = undefined;
			}
		}
	});

	$effect(() => {
		if (value) {
			const newInternalCalendarDate = dateToCalendarDate(value);
			if (!calendarDisplayDate || calendarDisplayDate.compare(newInternalCalendarDate) !== 0) {
				calendarDisplayDate = newInternalCalendarDate;
			}
		} else {
			if (calendarDisplayDate !== undefined) {
				calendarDisplayDate = undefined;
			}
		}
	});

	function handleCalendarInteraction(newDateValue?: DateValue) {
		open = false;
	}

	const df = new DateFormatter(getLocale(), {
		dateStyle: 'long'
	});
</script>

<div class="w-full" {...restProps}>
	<Popover.Root bind:open>
		<Popover.Trigger {id} class="w-full">
			{#snippet child({ props })}
				<Button
					{...props}
					{id}
					variant="outline"
					class={cn(
						'w-full justify-start text-left font-normal h-9 text-accent-foreground',
						!value && 'text-muted-foreground'
					)}
					aria-label={m.select_a_date()}
				>
					<CalendarIcon class="mr-2 size-4" />
					{calendarDisplayDate
						? df.format(calendarDisplayDate.toDate(getLocalTimeZone()))
						: m.select_a_date()}
				</Button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content class="w-auto p-0" align="start">
			<Calendar
				type="single"
				bind:value={calendarDisplayDate}
				onValueChange={handleCalendarInteraction}
				initialFocus
			/>
		</Popover.Content>
	</Popover.Root>
</div>

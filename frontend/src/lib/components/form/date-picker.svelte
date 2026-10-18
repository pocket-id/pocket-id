<script lang="ts">
	import { Calendar } from '#lib/components/ui/calendar/index.ts';
	import { filledFieldClass } from '#lib/components/ui/input/index.ts';
	import * as Popover from '#lib/components/ui/popover/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import { cn } from '#lib/utils/style.ts';
	import {
		CalendarDate,
		DateFormatter,
		getLocalTimeZone,
		today,
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
		calendarDisplayDate = newDateValue as CalendarDate | undefined;
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
	}

	const df = new DateFormatter(getLocale(), {
		dateStyle: 'long'
	});
</script>

<div class="w-full" {...restProps}>
	<Popover.Root bind:open>
		<Popover.Trigger {id} class="w-full">
			{#snippet child({ props })}
				<button
					{...props}
					class={cn(
						filledFieldClass,
						'text-body flex h-(--h-md) w-full items-center gap-2 rounded-sm px-3.5 max-sm:text-[16px]',
						!value && 'text-fg-muted'
					)}
					aria-label={m.select_a_date()}
				>
					<CalendarIcon class="text-fg-muted size-4 shrink-0" />
					{calendarDisplayDate
						? df.format(calendarDisplayDate.toDate(getLocalTimeZone()))
						: m.select_a_date()}
				</button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content class="w-auto p-0" align="start">
			<!-- Every picked date is an expiry that takes effect at the start of that day, so today is already too late -->
			<Calendar
				type="single"
				bind:value={() => calendarDisplayDate, (newValue) => handleCalendarInteraction(newValue)}
				minValue={today(getLocalTimeZone()).add({ days: 1 })}
				initialFocus
			/>
		</Popover.Content>
	</Popover.Root>
</div>

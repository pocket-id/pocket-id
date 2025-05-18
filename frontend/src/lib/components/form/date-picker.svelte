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

	let { value = $bindable(), ...restProps }: HTMLAttributes<HTMLButtonElement> & { value: Date } =
		$props();

	let date: CalendarDate = $state(dateToCalendarDate(value));
	let open = $state(false);

	function dateToCalendarDate(date: Date) {
		return new CalendarDate(date.getFullYear(), date.getMonth() + 1, date.getDate());
	}

	function onValueChange(newDate?: DateValue) {
		if (!newDate) return;

		value = newDate.toDate(getLocalTimeZone());
		date = newDate as CalendarDate;
		open = false;
	}

	const df = new DateFormatter(getLocale(), {
		dateStyle: 'long'
	});
</script>

<Popover.Root openFocus {open} onOpenChange={(o) => (open = o)}>
	<Popover.Trigger asChild>
		{#snippet children({ builder })}
			<Button
				{...restProps}
				variant="outline"
				class={cn('w-full justify-start text-left font-normal', !value && 'text-muted-foreground')}
				builders={[builder]}
			>
				<CalendarIcon class="mr-2 h-4 w-4" />
				{date ? df.format(date.toDate(getLocalTimeZone())) : m.select_a_date()}
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="w-auto p-0" align="start">
		<Calendar bind:value={date} initialFocus {onValueChange} />
	</Popover.Content>
</Popover.Root>

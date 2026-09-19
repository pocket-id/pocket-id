<script lang="ts">
	import * as ButtonGroup from '$lib/components/ui/button-group';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type { FormInput } from '$lib/utils/form-util';

	type DurationUnit = 'minutes' | 'hours' | 'days';

	const minutesPerUnit: Record<DurationUnit, number> = {
		minutes: 1,
		hours: 60,
		days: 24 * 60
	};
	const minimumMinutes = 1;
	const maximumMinutes = 365 * 24 * 60;

	let {
		id,
		label,
		description,
		input = $bindable()
	}: {
		id: string;
		label: string;
		description: string;
		input: FormInput<number>;
	} = $props();

	function preferredUnit(minutes: number): DurationUnit {
		if (minutes % minutesPerUnit.days === 0) return 'days';
		if (minutes % minutesPerUnit.hours === 0) return 'hours';
		return 'minutes';
	}

	function formatAmount(value: number): string {
		return Number(value.toFixed(10)).toString();
	}

	let unit = $state<DurationUnit>(preferredUnit(input.value));
	let amount = $state(formatAmount(input.value / minutesPerUnit[unit]));
	// The minutes that `amount` and `unit` currently represent. Only a value that arrives from
	// outside (e.g. after discarding changes) re-derives them, so that switching the unit isn't
	// undone by the rounding of the displayed amount.
	let displayedMinutes = input.value;

	$effect(() => {
		const minutes = input.value;
		if (Object.is(minutes, displayedMinutes)) return;

		displayedMinutes = minutes;
		if (!Number.isFinite(minutes)) return;
		unit = preferredUnit(minutes);
		amount = formatAmount(minutes / minutesPerUnit[unit]);
	});

	function updateAmount(event: Event) {
		amount = (event.currentTarget as HTMLInputElement).value;
		displayedMinutes = amount === '' ? Number.NaN : Number(amount) * minutesPerUnit[unit];
		input.value = displayedMinutes;
	}

	function updateUnit(value: string | undefined) {
		if (!value) return;

		unit = value as DurationUnit;
		if (Number.isFinite(input.value)) {
			amount = formatAmount(input.value / minutesPerUnit[unit]);
		}
	}

	function unitLabel(value: DurationUnit): string {
		switch (value) {
			case 'minutes':
				return m.minutes();
			case 'hours':
				return m.hours();
			case 'days':
				return m.days();
		}
	}
</script>

<Field.Field>
	<div>
		<Field.Label for={id}>{label}</Field.Label>
		<Field.Description>{description}</Field.Description>
	</div>
	<div>
		<ButtonGroup.Root class="w-full">
			<Input
				{id}
				type="number"
				value={amount}
				min={minimumMinutes / minutesPerUnit[unit]}
				max={maximumMinutes / minutesPerUnit[unit]}
				step={minimumMinutes / minutesPerUnit[unit]}
				aria-invalid={!!input.error}
				oninput={updateAmount}
			/>
			<Select.Root type="single" value={unit} onValueChange={updateUnit}>
				<Select.Trigger
					class="w-32"
					aria-label={m.duration_unit_for({ name: label })}
					aria-invalid={!!input.error}
				>
					{unitLabel(unit)}
				</Select.Trigger>
				<Select.Content>
					<Select.Group>
						{#each ['minutes', 'hours', 'days'] as option (option)}
							<Select.Item value={option}>{unitLabel(option as DurationUnit)}</Select.Item>
						{/each}
					</Select.Group>
				</Select.Content>
			</Select.Root>
		</ButtonGroup.Root>
		{#if input.error}
			<Field.Error>{input.error}</Field.Error>
		{/if}
	</div>
</Field.Field>

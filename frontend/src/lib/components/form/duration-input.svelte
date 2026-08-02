<script lang="ts">
	import * as ButtonGroup from '$lib/components/ui/button-group';
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type { FormInput } from '$lib/utils/form-util';

	type DurationUnit = 'minutes' | 'hours' | 'days';

	const secondsPerUnit: Record<DurationUnit, number> = {
		minutes: 60,
		hours: 60 * 60,
		days: 24 * 60 * 60
	};
	const minimumSeconds = 60;
	const maximumSeconds = 365 * 24 * 60 * 60;

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

	function preferredUnit(seconds: number): DurationUnit {
		if (seconds % secondsPerUnit.days === 0) return 'days';
		if (seconds % secondsPerUnit.hours === 0) return 'hours';
		return 'minutes';
	}

	function formatAmount(value: number): string {
		return Number(value.toFixed(10)).toString();
	}

	let unit = $state<DurationUnit>(preferredUnit(input.value));
	let amount = $state(formatAmount(input.value / secondsPerUnit[unit]));

	function updateAmount(event: Event) {
		amount = (event.currentTarget as HTMLInputElement).value;
		input.value = amount === '' ? Number.NaN : Number(amount) * secondsPerUnit[unit];
	}

	function updateUnit(value: string | undefined) {
		if (!value) return;

		unit = value as DurationUnit;
		if (Number.isFinite(input.value)) {
			amount = formatAmount(input.value / secondsPerUnit[unit]);
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
				min={minimumSeconds / secondsPerUnit[unit]}
				max={maximumSeconds / secondsPerUnit[unit]}
				step={minimumSeconds / secondsPerUnit[unit]}
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

<script lang="ts">
	import * as Field from '$lib/components/ui/field';
	import { Input } from '$lib/components/ui/input';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { m } from '$lib/paraglide/messages';
	import type { CustomField, CustomFieldValue } from '$lib/types/custom-field.type';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		customFieldValues = $bindable(),
		customFields = []
	}: HTMLAttributes<HTMLDivElement> & {
		customFieldValues: CustomFieldValue[];
		customFields?: CustomField[];
	} = $props();

	let errors = $state<Record<string, string>>({});

	function getValue(field: CustomField) {
		const customFieldValue = customFieldValues.find(
			(customFieldValue) => customFieldValue.customFieldId === field.id
		);
		if (customFieldValue) return customFieldValue.value;

		return field.defaultValue ?? '';
	}

	function setValue(field: CustomField, value: string) {
		errors = { ...errors, [field.key]: '' };
		const nextCustomFieldValues = customFieldValues.filter(
			(customFieldValue) => customFieldValue.customFieldId !== field.id
		);
		if (field.type !== 'boolean' && !field.required && value === '') {
			customFieldValues = nextCustomFieldValues;
			return;
		}
		customFieldValues = [...nextCustomFieldValues, { customFieldId: field.id, value }];
	}

	function normalizeCustomFieldValues() {
		const normalizedCustomFieldValues: CustomFieldValue[] = [];
		for (const field of customFields) {
			const value = field.type === 'boolean' ? getValue(field) || 'false' : getValue(field);
			if (field.type !== 'boolean' && !field.required && value === '') continue;
			normalizedCustomFieldValues.push({ customFieldId: field.id, value });
		}
		customFieldValues = normalizedCustomFieldValues;
	}

	export function validate() {
		normalizeCustomFieldValues();
		const nextErrors: Record<string, string> = {};

		for (const field of customFields) {
			const value = getValue(field);
			if (field.required && field.type !== 'boolean' && value === '') {
				nextErrors[field.key] = m.field_is_required();
				continue;
			}
			if (field.type === 'number' && value !== '' && Number.isNaN(Number(value))) {
				nextErrors[field.key] = m.must_be_a_number();
				continue;
			}
			if (field.type === 'string' && field.validationRegex && value !== '') {
				let regex: RegExp;
				try {
					regex = new RegExp(field.validationRegex);
				} catch {
					nextErrors[field.key] = m.invalid_regex();
					continue;
				}
				if (!regex.test(value)) {
					nextErrors[field.key] =
						field.validationErrorMessage || m.value_does_not_match_required_format();
				}
			}
		}

		errors = nextErrors;
		return Object.keys(nextErrors).length === 0;
	}
</script>

{#each customFields as field (field.key)}
	<Field.Field>
		<Field.Label required={field.required} for={`custom-field-${field.key}`}>
			{field.displayName || field.key}
		</Field.Label>
		{#if field.type === 'boolean'}
			<div class="flex h-9 items-center">
				<Switch
					id={`custom-field-${field.key}`}
					checked={getValue(field) === 'true'}
					onCheckedChange={(checked) => setValue(field, checked ? 'true' : 'false')}
				/>
			</div>
		{:else}
			<Input
				id={`custom-field-${field.key}`}
				type={field.type === 'number' ? 'number' : 'text'}
				value={getValue(field)}
				aria-invalid={!!errors[field.key]}
				oninput={(e) => setValue(field, e.currentTarget.value)}
			/>
		{/if}
		{#if errors[field.key]}
			<Field.Error class="text-start">{errors[field.key]}</Field.Error>
		{/if}
	</Field.Field>
{/each}

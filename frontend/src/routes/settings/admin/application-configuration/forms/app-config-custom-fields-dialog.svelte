<script lang="ts">
	import FormInput from '$lib/components/form/form-input.svelte';
	import SwitchWithLabel from '$lib/components/form/switch-with-label.svelte';
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { m } from '$lib/paraglide/messages';
	import type {
		CustomField,
		CustomFieldTarget,
		CustomFieldType
	} from '$lib/types/custom-field.type';
	import { preventDefault } from '$lib/utils/event-util';
	import { createForm } from '$lib/utils/form-util';
	import { toast } from 'svelte-sonner';
	import z from 'zod';

	let {
		show = $bindable(),
		existingCustomField,
		callback
	}: {
		show?: boolean;
		existingCustomField?: CustomField;
		callback: (updatedField: CustomField) => Promise<void>;
	} = $props();

	let isLoading = $state(false);

	type CustomFieldForm = CustomField & {
		defaultValue: string;
		validationRegex: string;
		validationErrorMessage: string;
	};

	const customField: CustomFieldForm = {
		id: '',
		key: '',
		displayName: '',
		type: 'string',
		target: 'user',
		required: false,
		userEditable: false,
		defaultValue: '',
		validationRegex: '',
		validationErrorMessage: ''
	};

	const fieldTypes: { value: CustomFieldType; label: string }[] = [
		{ value: 'string', label: m.text() },
		{ value: 'number', label: m.number() },
		{ value: 'boolean', label: m.boolean() }
	];
	const fieldTargets: { value: CustomFieldTarget; label: string }[] = [
		{ value: 'user', label: m.users() },
		{ value: 'group', label: m.groups() },
		{ value: 'both', label: m.users_and_groups() }
	];

	const formSchema = z
		.object({
			id: z.string().min(1),
			key: z.string().trim().min(1, { message: m.field_is_required() }),
			displayName: z.string().trim().min(1, { message: m.field_is_required() }),
			type: z.enum(['string', 'number', 'boolean']),
			target: z.enum(['user', 'group', 'both']),
			required: z.boolean(),
			userEditable: z.boolean(),
			defaultValue: z.preprocess(
				(value) => (value === null || value === undefined ? '' : String(value)),
				z.string()
			),
			validationRegex: z.string().refine(
				(value) => {
					if (!value) return true;
					try {
						new RegExp(value);
						return true;
					} catch {
						return false;
					}
				},
				{ message: m.invalid_regex() }
			),
			validationErrorMessage: z.string()
		})
		.superRefine((data, ctx) => {
			if (data.required && !data.defaultValue) {
				ctx.addIssue({
					code: 'custom',
					path: ['defaultValue'],
					message: m.field_is_required()
				});
				return;
			}
			if (!data.defaultValue) return;

			if (data.type === 'number' && Number.isNaN(Number(data.defaultValue))) {
				ctx.addIssue({
					code: 'custom',
					path: ['defaultValue'],
					message: m.must_be_a_number()
				});
			}
			if (data.type === 'string' && data.validationRegex) {
				try {
					const regex = new RegExp(data.validationRegex);
					if (!regex.test(data.defaultValue)) {
						ctx.addIssue({
							code: 'custom',
							path: ['defaultValue'],
							message: data.validationErrorMessage || m.value_does_not_match_required_format()
						});
					}
				} catch {}
			}
		});
	type FormSchema = typeof formSchema;

	const { inputs, ...form } = createForm<FormSchema>(formSchema, customField);

	$effect(() => {
		if (!show) return;

		const field = existingCustomField ?? {
			...customField,
			id: crypto.randomUUID()
		};

		form.setValue('id', field.id);
		form.setValue('key', field.key);
		form.setValue('displayName', field.displayName);
		form.setValue('type', field.type);
		form.setValue('target', field.target || 'both');
		form.setValue('required', field.required);
		form.setValue('userEditable', !!field.userEditable);
		form.setValue('defaultValue', field.defaultValue || '');
		form.setValue('validationRegex', field.validationRegex || '');
		form.setValue('validationErrorMessage', field.validationErrorMessage || '');
	});

	async function onSubmit() {
		isLoading = true;
		try {
			const data = form.validate();
			if (!data) return;

			await callback(data);
			toast.success(m.custom_fields_updated_successfully());
			show = false;
		} finally {
			isLoading = false;
		}
	}
</script>

<Dialog.Root open={show} onOpenChange={(open) => (show = open)}>
	<Dialog.Content class="lg:min-w-3xl md:min-w-2xl" onOpenAutoFocus={(e) => e.preventDefault()}>
		<Dialog.Header>
			<Dialog.Title>{existingCustomField ? m.edit() : m.add_custom_field()}</Dialog.Title>
		</Dialog.Header>
		<form onsubmit={preventDefault(onSubmit)}>
			<div class="grid grid-cols-1 items-center gap-5 md:grid-cols-2">
				<FormInput label={m.key()} bind:input={$inputs.key} />
				<FormInput label={m.display_name()} bind:input={$inputs.displayName} />
				<Field.Field>
					<Field.Label>{m.type()}</Field.Label>
					<Select.Root
						type="single"
						disabled={!!existingCustomField}
						value={$inputs.type.value}
						onValueChange={(value) => {
							$inputs.type.value = value as any;
							if (value !== 'string') {
								$inputs.validationRegex.value = '';
								$inputs.validationErrorMessage.value = '';
							}
						}}
					>
						<Select.Trigger class="w-full" aria-label={m.type()}>
							{fieldTypes.find((option) => option.value === $inputs.type.value)?.label}
						</Select.Trigger>
						<Select.Content>
							{#each fieldTypes as option}
								<Select.Item value={option.value}>{option.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</Field.Field>
				<Field.Field>
					<Field.Label>{m.available_for()}</Field.Label>
					<Select.Root
						type="single"
						value={$inputs.target.value}
						onValueChange={(value) => ($inputs.target.value = value as CustomFieldTarget)}
					>
						<Select.Trigger class="w-full" aria-label={m.available_for()}>
							{fieldTargets.find((option) => option.value === $inputs.target.value)?.label}
						</Select.Trigger>
						<Select.Content>
							{#each fieldTargets as option}
								<Select.Item value={option.value}>{option.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</Field.Field>
				<SwitchWithLabel
					id="required"
					class="items-center"
					label={m.required()}
					description={m.required_custom_field_description()}
					bind:checked={$inputs.required.value}
				/>
				{#if $inputs.target.value != 'group'}
					<SwitchWithLabel
						id="user-editable"
						class="items-center"
						label={m.user_editable()}
						description={m.user_editable_description()}
						bind:checked={$inputs.userEditable.value}
					/>
					{:else}
					<div class="hidden md:flex"></div>
				{/if}
				{#if $inputs.type.value === 'boolean'}
					<Field.Field>
						<Field.Label required={$inputs.required.value}>{m.default_value()}</Field.Label>
						<Select.Root
							type="single"
							value={$inputs.defaultValue.value || ''}
							onValueChange={(value) => ($inputs.defaultValue.value = value)}
						>
							<Select.Trigger class="w-full" aria-label={m.default_value()}>
								{#if $inputs.defaultValue.value === 'true'}
									{m.yes()}
								{:else if $inputs.defaultValue.value === 'false'}
									{m.no()}
								{:else}
									{m.no_default()}
								{/if}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="">{m.no_default()}</Select.Item>
								<Select.Item value="true">{m.yes()}</Select.Item>
								<Select.Item value="false">{m.no()}</Select.Item>
							</Select.Content>
						</Select.Root>
						{#if $inputs.defaultValue.error}
							<Field.Error class="text-start">{$inputs.defaultValue.error}</Field.Error>
						{/if}
					</Field.Field>
				{:else}
					<FormInput
						label={m.default_value()}
						type={$inputs.type.value === 'number' ? 'number' : 'text'}
						bind:input={$inputs.defaultValue}
					/>
				{/if}
				{#if $inputs.type.value === 'string'}
					<FormInput
						label={m.validation_regex()}
						placeholder="^\S+@\S+\.\S+$"
						bind:input={$inputs.validationRegex}
						description={m.validation_regex_description()}
					/>
					<FormInput
						label={m.validation_error_message()}
						placeholder={m.must_be_a_valid_email_address()}
						bind:input={$inputs.validationErrorMessage}
						description={m.validation_error_message_description()}
					/>
				{/if}
			</div>
			<Dialog.Footer class="mt-10 md:mt-3">
				<Button onclick={() => (show = false)} variant="secondary">{m.cancel()}</Button>
				<Button {isLoading} type="submit">{m.save()}</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

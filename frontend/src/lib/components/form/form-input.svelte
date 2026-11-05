<script lang="ts">
	import EnvOverrideLock from '#lib/components/env-override-lock.svelte';
	import DatePicker from '#lib/components/form/date-picker.svelte';
	import * as Field from '@pocket-id/ui/field';
	import { Input, type FormInputEvent } from '@pocket-id/ui/input';
	import { m } from '#lib/paraglide/messages.js';
	import type { FormInput } from '#lib/utils/form-util.ts';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes, HTMLInputAttributes } from 'svelte/elements';

	type WithoutChildren = {
		children?: undefined;
		input?: FormInput<string | boolean | number | Date | undefined>;
		labelFor?: never;
	};
	type WithChildren = {
		children: Snippet;
		input?: any;
		labelFor?: string;
	};

	let {
		input = $bindable(),
		label,
		description,
		placeholder,
		autocomplete,
		disabled = false,
		type = 'text',
		children,
		onInput,
		labelFor,
		readonly = false,
		inputClass,
		changed = false,
		optional = false,
		envOverride = false,
		class: className,
		...restProps
	}: HTMLAttributes<HTMLDivElement> &
		(WithChildren | WithoutChildren) & {
			label?: string;
			description?: string | Snippet;
			placeholder?: string;
			autocomplete?: HTMLInputAttributes['autocomplete'];
			disabled?: boolean;
			inputClass?: string;
			type?: 'text' | 'password' | 'email' | 'number' | 'checkbox' | 'date' | 'url';
			readonly?: boolean;
			changed?: boolean;
			optional?: boolean;
			// Makes the input read-only and marks it as set by an environment variable
			envOverride?: boolean;
			onInput?: (e: FormInputEvent) => void;
		} = $props();

	const id = label?.toLowerCase().replace(/ /g, '-');
</script>

<Field.Field data-disabled={disabled || envOverride} class={className} {...restProps}>
	{#if label}
		<div class="flex items-center gap-2">
			<Field.Label optional={optional && !readonly} for={labelFor ?? id}>{label}</Field.Label>
			<!-- Outside the label, so the changed state doesn't become part of the input's name -->
			{#if changed}
				<span class="bg-fg size-1.5 shrink-0 rounded-full"></span>
				<span class="sr-only">{m.changed_from_the_default()}</span>
			{/if}
			{#if envOverride}
				<EnvOverrideLock />
			{/if}
		</div>
	{/if}
	<div class="flex flex-col gap-1.5">
		{#if children}
			{@render children()}
		{:else if input}
			{#if type === 'date'}
				<DatePicker
					{id}
					bind:value={
						() => input.value as Date | undefined,
						(value) => {
							input.value = value;
							input.error = null;
						}
					}
				/>
			{:else}
				<Input
					aria-invalid={!!input.error}
					class={inputClass}
					{id}
					{placeholder}
					{type}
					{autocomplete}
					bind:value={
						() => input.value as string | number | undefined,
						(value) => {
							input.value = value;
							input.error = null;
						}
					}
					disabled={disabled || envOverride}
					oninput={(e) => onInput?.(e)}
					{readonly}
				/>
			{/if}
		{/if}
		{#if input?.error}
			<Field.Error class="text-start">{input.error}</Field.Error>
		{:else if description}
			<Field.Description>
				{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
			</Field.Description>
		{/if}
	</div>
</Field.Field>

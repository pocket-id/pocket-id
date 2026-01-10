<script lang="ts">
	import DatePicker from '$lib/components/form/date-picker.svelte';
	import * as Field from '$lib/components/ui/field';
	import { Input, type FormInputEvent } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import type { FormInput } from '$lib/utils/form-util';
	import { LucideExternalLink } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import FormattedMessage from '../formatted-message.svelte';

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
		docsLink,
		placeholder,
		disabled = false,
		type = 'text',
		children,
		onInput,
		labelFor,
		inputClass,
		...restProps
	}: HTMLAttributes<HTMLDivElement> &
		(WithChildren | WithoutChildren) & {
			label?: string;
			description?: string;
			docsLink?: string;
			placeholder?: string;
			disabled?: boolean;
			inputClass?: string;
			type?: 'text' | 'password' | 'email' | 'number' | 'checkbox' | 'date';
			onInput?: (e: FormInputEvent) => void;
		} = $props();

	const id = label?.toLowerCase().replace(/ /g, '-');
</script>

<Field.Field data-disabled={disabled} {...restProps}>
	{#if label}
		<Field.Label required={input?.required} class="mb-0" for={labelFor ?? id}>{label}</Field.Label>
	{/if}
	{#if description}
		<Field.Description>
			<FormattedMessage m={description} />
			{#if docsLink}
				<a
					class="relative text-black after:absolute after:bottom-0 after:left-0 after:h-px after:w-full after:translate-y-[-1px] after:bg-white dark:text-white"
					href={docsLink}
					target="_blank"
				>
					{m.docs()}
					<LucideExternalLink class="inline size-3 align-text-top" />
				</a>
			{/if}
		</Field.Description>
	{/if}
	{#if children}
		{@render children()}
	{:else if input}
		{#if type === 'date'}
			<DatePicker {id} bind:value={input.value as Date} />
		{:else}
			<Input
				aria-invalid={!!input.error}
				class={inputClass}
				{id}
				{placeholder}
				{type}
				bind:value={input.value}
				{disabled}
				oninput={(e) => onInput?.(e)}
			/>
		{/if}
	{/if}
	{#if input?.error}
		<Field.Error>{input.error}</Field.Error>
	{/if}
</Field.Field>

<script lang="ts">
	import DatePicker from '$lib/components/form/date-picker.svelte';
	import * as Field from '$lib/components/ui/field';
	import { Input, type FormInputEvent } from '$lib/components/ui/input';
	import { m } from '$lib/paraglide/messages';
	import type { FormInput } from '$lib/utils/form-util';
	import { LucideExternalLink } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

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
		...restProps
	}: HTMLAttributes<HTMLDivElement> & {
		input?: FormInput<string | boolean | number | Date | undefined>;
		label?: string;
		description?: string;
		docsLink?: string;
		placeholder?: string;
		disabled?: boolean;
		type?: 'text' | 'password' | 'email' | 'number' | 'checkbox' | 'date';
		onInput?: (e: FormInputEvent) => void;
		children?: Snippet;
	} = $props();

	const id = label?.toLowerCase().replace(/ /g, '-');
</script>

<Field.Field data-disabled={disabled} {...restProps}>
	{#if label}
		<Field.Label required={input?.required} for={id}>{label}</Field.Label>
	{/if}
	{#if description}
		<Field.Description>
			{description}
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

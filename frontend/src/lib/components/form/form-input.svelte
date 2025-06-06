<script lang="ts">
	import DatePicker from '$lib/components/form/date-picker.svelte';
	import { Input, type FormInputEvent } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import type { FormInput } from '$lib/utils/form-util';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		input = $bindable(),
		label,
		description,
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
		placeholder?: string;
		disabled?: boolean;
		type?: 'text' | 'password' | 'email' | 'number' | 'checkbox' | 'date';
		onInput?: (e: FormInputEvent) => void;
		children?: Snippet;
	} = $props();

	const id = label?.toLowerCase().replace(/ /g, '-');
</script>

<div {...restProps}>
	{#if label}
		<Label class="mb-0" for={id}>{label}</Label>
	{/if}
	{#if description}
		<p class="text-muted-foreground mt-1 text-xs">{description}</p>
	{/if}
	<div class={label || description ? 'mt-2' : ''}>
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
			<p class="text-destructive mt-1 text-xs">{input.error}</p>
		{/if}
	</div>
</div>

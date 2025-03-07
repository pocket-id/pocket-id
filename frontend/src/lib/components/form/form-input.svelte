<script lang="ts">
	import { Label } from '$lib/components/ui/label';
	import type { FormInput } from '$lib/utils/form-util';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import { Input, type FormInputEvent } from '$lib/components/ui/input';

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
		input?: FormInput<string | boolean | number>;
		label?: string;
		description?: string;
		placeholder?: string;
		disabled?: boolean;
		type?: 'text' | 'password' | 'email' | 'number' | 'checkbox' | 'datetime-local';
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
			<Input
				{id}
				{placeholder}
				{type}
				bind:value={input.value}
				{disabled}
				on:input={(e) => onInput?.(e)}
			/>
		{/if}
		{#if input?.error}
			<p class="mt-1 text-sm text-red-500">{input.error}</p>
		{/if}
	</div>
</div>

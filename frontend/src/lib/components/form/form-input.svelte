<script lang="ts">
	import DatePicker from '$lib/components/form/date-picker.svelte';
	import { Input, type FormInputEvent } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { m } from '$lib/paraglide/messages';
	import type { FormInput } from '$lib/utils/form-util';
	import { LucideExternalLink } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';
	import FormattedMessage from '../formatted-message.svelte';

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

<div {...restProps}>
	{#if label}
		<Label required={input?.required} class="mb-0" for={id}>{label}</Label>
	{/if}
	{#if description}
		<p class="text-muted-foreground mt-1 text-xs">
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
		</p>
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
			<p class="text-destructive mt-1 text-start text-xs">{input.error}</p>
		{/if}
	</div>
</div>

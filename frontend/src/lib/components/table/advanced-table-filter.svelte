<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import XIcon from '@lucide/svelte/icons/x';

	let {
		title,
		options,
		value,
		onChange
	}: {
		title: string;
		options: { label: string; value: string | boolean }[];
		value?: string | boolean;
		onChange: (value?: string | boolean) => void;
	} = $props();

	// Select values are strings, so "Any" needs a sentinel that no option value uses
	const ANY = '__any';

	const testId = $derived(`facet-${title.toLowerCase()}`);
	const selected = $derived(options.find((o) => o.value === value));
	const valueLabel = $derived(selected?.label ?? m.any());
</script>

<span class="inline-flex shrink-0 items-center gap-0.5">
	<Select.Root
		type="single"
		value={selected ? String(selected.value) : ANY}
		onValueChange={(v) => onChange(options.find((o) => String(o.value) === v)?.value)}
	>
		<Select.Trigger
			variant="ghost"
			aria-label={m.facet_filter_label({ name: title, value: valueLabel })}
			data-testid={`${testId}-trigger`}
		>
			<span class="text-fg-muted">{title}:</span>
			<span class={selected ? 'text-fg font-medium' : 'text-fg-muted'}>{valueLabel}</span>
		</Select.Trigger>
		<Select.Content align="start">
			<Select.Item value={ANY} label={m.any()} data-testid={`${testId}-option-any`} />
			{#each options as option (option.value)}
				<Select.Item
					value={String(option.value)}
					label={option.label}
					data-testid={`${testId}-option-${String(option.value)}`}
				/>
			{/each}
		</Select.Content>
	</Select.Root>
	{#if selected}
		<Button
			variant="ghost"
			size="icon-xs"
			class="size-5 max-sm:hidden"
			aria-label={m.clear_name_filter({ name: title })}
			onclick={() => onChange(undefined)}
		>
			<XIcon class="size-3" />
		</Button>
	{/if}
</span>

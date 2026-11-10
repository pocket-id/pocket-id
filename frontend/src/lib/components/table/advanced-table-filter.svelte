<script lang="ts">
	import { Button } from '@pocket-id/ui/button';
	import * as Command from '@pocket-id/ui/command';
	import * as Popover from '@pocket-id/ui/popover';
	import * as Select from '@pocket-id/ui/select';
	import { m } from '#lib/paraglide/messages.js';
	import CheckIcon from '~icons/solar/check-linear';
	import ChevronDownIcon from '~icons/solar/alt-arrow-down-linear';
	import XIcon from '~icons/solar/close-linear';

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

	// Longer lists get a search field, which Select can't hold
	const SEARCH_THRESHOLD = 10;

	const testId = $derived(`facet-${title.toLowerCase()}`);
	const selected = $derived(options.find((o) => o.value === value));
	const valueLabel = $derived(selected?.label ?? m.any());

	let open = $state(false);
	let search = $state('');
	const filteredOptions = $derived(
		options.filter((o) => o.label.toLowerCase().includes(search.trim().toLowerCase()))
	);

	function choose(value?: string | boolean) {
		onChange(value);
		open = false;
	}
</script>

{#snippet triggerLabel()}
	<span class="text-fg-muted">{title}:</span>
	<span class={selected ? 'text-fg font-medium' : 'text-fg-muted'}>{valueLabel}</span>
{/snippet}

{#snippet checkSlot(checked: boolean)}
	<span class="flex size-4 items-center justify-center">
		{#if checked}<CheckIcon class="text-primary-control" />{/if}
	</span>
{/snippet}

<span class="inline-flex shrink-0 items-center gap-0.5">
	{#if options.length > SEARCH_THRESHOLD}
		<Popover.Root bind:open onOpenChange={(o) => o && (search = '')}>
			<Popover.Trigger
				aria-label={m.facet_filter_label({ name: title, value: valueLabel })}
				data-testid={`${testId}-trigger`}
				class="hover:bg-fill-hover aria-expanded:bg-fill-selected focus-visible:outline-ring text-control flex h-7 items-center gap-1 rounded-xs pr-1.5 pl-2 whitespace-nowrap transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-2"
			>
				{@render triggerLabel()}
				<ChevronDownIcon class="text-fg-muted size-3" />
			</Popover.Trigger>
			<Popover.Content align="start" class="w-64 p-0">
				<Command.Root shouldFilter={false}>
					<Command.Input bind:value={search} placeholder={m.search()} />
					<Command.List class="mt-1">
						<Command.Empty>{m.no_items_found()}</Command.Empty>
						{#if !search.trim()}
							<Command.Item
								value={ANY}
								onSelect={() => choose(undefined)}
								data-testid={`${testId}-option-any`}
							>
								{@render checkSlot(!selected)}
								{m.any()}
							</Command.Item>
						{/if}
						{#each filteredOptions as option (option.value)}
							<Command.Item
								value={String(option.value)}
								onSelect={() => choose(option.value)}
								data-testid={`${testId}-option-${String(option.value)}`}
							>
								{@render checkSlot(option === selected)}
								<span class="truncate">{option.label}</span>
							</Command.Item>
						{/each}
					</Command.List>
				</Command.Root>
			</Popover.Content>
		</Popover.Root>
	{:else}
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
				{@render triggerLabel()}
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
	{/if}
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

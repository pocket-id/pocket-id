<script lang="ts" generics="TData, TValue">
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils/style';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ListFilterIcon from '@lucide/svelte/icons/list-filter';
	import type { Component } from 'svelte';

	let {
		title,
		options,
		selectedValues = new Set<string | boolean>(),
		showCheckboxes = true,
		onChanged = (selected: Set<string | boolean>) => {}
	}: {
		title: string;
		options: {
			label: string;
			value: string | boolean;
			icon?: Component;
		}[];
		selectedValues?: Set<string | boolean>;
		showCheckboxes?: boolean;
		onChanged?: (selected: Set<string | boolean>) => void;
	} = $props();
</script>

<Popover.Root>
	<Popover.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="outline"
				size="sm"
				class="h-8 border-dashed"
				data-testid={`facet-${title.toLowerCase()}-trigger`}
			>
				<ListFilterIcon />
				{title}
				{#if selectedValues.size > 0}
					<Separator orientation="vertical" class="mx-2 h-4" />
					<Badge variant="secondary" class="rounded-sm px-1 font-normal lg:hidden">
						{selectedValues.size}
					</Badge>
					<div class="hidden space-x-1 lg:flex">
						{#if selectedValues.size > 2}
							<Badge variant="secondary" class="rounded-sm px-1 font-normal">
								Count: {selectedValues.size}
							</Badge>
						{:else}
							{#each options.filter((opt) => selectedValues.has(opt.value)) as option (option)}
								<Badge variant="secondary" class="rounded-sm px-1 font-normal">
									{option.label}
								</Badge>
							{/each}
						{/if}
					</div>
				{/if}
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content
		class="w-[200px] p-0"
		align="start"
		data-testid={`facet-${title.toLowerCase()}-content`}
	>
		<Command.Root>
			<Command.Input placeholder={title} />
			<Command.List>
				<Command.Empty>{m.no_items_found()}</Command.Empty>
				<Command.Group>
					{#each options as option (option)}
						{@const isSelected = selectedValues.has(option.value)}
						<Command.Item
							data-testid={`facet-${title.toLowerCase()}-option-${String(option.value)}`}
							onSelect={() => {
								if (isSelected) {
									selectedValues = new Set([...selectedValues].filter((v) => v !== option.value));
								} else {
									selectedValues = new Set([...selectedValues, option.value]);
								}
								onChanged(selectedValues);
							}}
						>
							{#if showCheckboxes}
								<div
									class={cn(
										'border-primary mr-2 flex size-4 items-center justify-center rounded-sm border',
										isSelected
											? 'bg-primary text-primary-foreground'
											: 'opacity-50 [&_svg]:invisible'
									)}
								>
									<CheckIcon class="size-4" />
								</div>
							{/if}
							{#if option.icon}
								{@const Icon = option.icon}
								<Icon class="text-muted-foreground" />
							{/if}

							<span>{option.label}</span>
						</Command.Item>
					{/each}
				</Command.Group>
				{#if selectedValues.size > 0}
					<Command.Separator />
					<Command.Group>
						<Command.Item onSelect={() => {}} class="justify-center text-center">
							Clear filters
						</Command.Item>
					</Command.Group>
				{/if}
			</Command.List>
		</Command.Root>
	</Popover.Content>
</Popover.Root>

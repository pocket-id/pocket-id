<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils/style';
	import { LucideChevronDown, type Icon as IconType } from '@lucide/svelte';
	import { onMount, type Snippet } from 'svelte';
	import { slide } from 'svelte/transition';
	import { Button } from './ui/button';
	import * as Card from './ui/card';

	let {
		id,
		title,
		description,
		defaultExpanded = false,
		forcedExpanded,
		button,
		icon,
		children
	}: {
		id: string;
		title: string;
		description?: string;
		defaultExpanded?: boolean;
		forcedExpanded?: boolean;
		icon?: typeof IconType;
		button?: Snippet;
		children: Snippet;
	} = $props();

	let expanded = $state(defaultExpanded);

	function loadExpandedState() {
		const state = JSON.parse(localStorage.getItem('collapsible-cards-expanded') || '{}');
		expanded = state[id] || false;
	}

	function saveExpandedState() {
		const state = JSON.parse(localStorage.getItem('collapsible-cards-expanded') || '{}');
		state[id] = expanded;
		localStorage.setItem('collapsible-cards-expanded', JSON.stringify(state));
	}

	function toggleExpanded() {
		expanded = !expanded;
		saveExpandedState();
	}

	onMount(() => {
		if (defaultExpanded) {
			saveExpandedState();
		}
		loadExpandedState();
	});

	$effect(() => {
		if (forcedExpanded !== undefined) {
			expanded = forcedExpanded;
		}
	});
</script>

<Card.Root>
	<Card.Header class="bg-card cursor-pointer" onclick={toggleExpanded}>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					{#if icon}{@const Icon = icon}
						<Icon class="text-primary/80 size-5" />
					{/if}
					{title}
				</Card.Title>
				{#if description}
					<Card.Description>{description}</Card.Description>
				{/if}
			</div>
			{#if button}
				{@render button()}
			{:else}
				<Button class="ml-10 h-8 p-3" variant="ghost" aria-label={m.expand_card()}>
					<LucideChevronDown
						class={cn(
							'size-5 transition-transform duration-200',
							expanded && 'rotate-180 transform'
						)}
					/>
				</Button>
			{/if}
		</div>
	</Card.Header>
	{#if expanded}
		<div transition:slide={{ duration: 200 }}>
			<Card.Content>
				{@render children()}
			</Card.Content>
		</div>
	{/if}
</Card.Root>

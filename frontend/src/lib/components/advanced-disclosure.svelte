<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideChevronDown } from '@lucide/svelte';
	import { Collapsible } from 'bits-ui';
	import { untrack, type Snippet } from 'svelte';

	let {
		title = m.advanced(),
		summary,
		changedCount = 0,
		children
	}: {
		title?: string;
		summary: string;
		changedCount?: number;
		children: Snippet;
	} = $props();

	const titleId = $props.id();

	// Only untouched defaults may hide, so a changed setting opens the disclosure on load
	let open = $state(untrack(() => changedCount > 0));
</script>

<Collapsible.Root bind:open>
	{#snippet child({ props })}
		<section {...props} aria-labelledby={titleId} class="flex flex-col gap-6">
			<Collapsible.Trigger
				class="group/advanced focus-visible:outline-ring flex w-full flex-col gap-1 rounded-xs text-start focus-visible:outline-2 focus-visible:outline-offset-2"
			>
				<span class="flex w-full items-center gap-3">
					<span id={titleId} class="text-title text-fg">{title}</span>
					{#if changedCount > 0}
						<Badge>{m.count_changed({ count: changedCount })}</Badge>
					{/if}
					<span aria-hidden="true" class="bg-border h-px flex-1"></span>
					<LucideChevronDown
						class="text-fg-muted size-4 shrink-0 transition-transform duration-(--dur-base) ease-in-out group-data-[state=open]/advanced:rotate-180"
					/>
				</span>
				<span class="text-caption text-fg-muted">{summary}</span>
			</Collapsible.Trigger>
			<Collapsible.Content
				class="*:transition-motion flex flex-col gap-10 *:duration-(--dur-base) data-starting-style:*:-translate-y-(--rise) data-starting-style:*:opacity-0"
			>
				{@render children()}
			</Collapsible.Content>
		</section>
	{/snippet}
</Collapsible.Root>

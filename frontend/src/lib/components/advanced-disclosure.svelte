<script lang="ts">
	import { Badge } from '@pocket-id/ui/badge';
	import { cn } from '@pocket-id/ui/utils';
	import { m } from '#lib/paraglide/messages.js';
	import ChevronDownIcon from '~icons/solar/alt-arrow-down-linear';
	import { Collapsible } from 'bits-ui';
	import { untrack, type Snippet } from 'svelte';

	let {
		title = m.advanced(),
		titleSize = 'title',
		summary,
		changedCount = 0,
		// Only untouched defaults may hide, so a changed setting opens the disclosure on load
		open = $bindable(untrack(() => changedCount > 0)),
		children
	}: {
		title?: string;
		// A disclosure inside a page section takes the section heading size instead of the page-level title size
		titleSize?: 'title' | 'heading';
		summary?: string;
		changedCount?: number;
		open?: boolean;
		children: Snippet;
	} = $props();

	const titleId = $props.id();
</script>

<Collapsible.Root bind:open>
	{#snippet child({ props })}
		<section {...props} aria-labelledby={titleId} class="flex flex-col gap-6">
			<Collapsible.Trigger
				class="group/advanced focus-visible:outline-ring flex w-full flex-col gap-1 rounded-xs text-start focus-visible:outline-2 focus-visible:outline-offset-2"
			>
				<span class="flex w-full items-center gap-3">
					<span
						id={titleId}
						class={cn('text-fg', titleSize === 'title' ? 'text-title' : 'text-heading')}
					>
						{title}
					</span>
					{#if changedCount > 0}
						<Badge>{m.count_changed({ count: changedCount })}</Badge>
					{/if}
					<span aria-hidden="true" class="bg-border h-px flex-1"></span>
					<ChevronDownIcon
						class="text-fg-muted size-4 shrink-0 transition-transform duration-(--dur-base) ease-in-out group-data-[state=open]/advanced:rotate-180"
					/>
				</span>
				{#if summary}
					<span class="text-caption text-fg-muted">{summary}</span>
				{/if}
			</Collapsible.Trigger>
			<Collapsible.Content
				class="*:transition-motion flex flex-col gap-10 *:duration-(--dur-base) data-starting-style:*:-translate-y-(--rise) data-starting-style:*:opacity-0"
			>
				{@render children()}
			</Collapsible.Content>
		</section>
	{/snippet}
</Collapsible.Root>

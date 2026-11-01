<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideChevronDown } from '@lucide/svelte';
	import { Collapsible } from 'bits-ui';
	import { untrack, type Snippet } from 'svelte';

	let {
		summary,
		changedCount = 0,
		children
	}: {
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
		<section {...props} aria-labelledby={titleId} class="flex flex-col gap-10">
			<Collapsible.Trigger
				class="group/advanced bg-fill-muted text-control text-fg before:bg-fill-hover focus-visible:outline-ring relative isolate flex min-h-14 w-full items-center gap-4 rounded-lg px-4 py-3 text-start before:absolute before:inset-0 before:-z-1 before:rounded-[inherit] before:opacity-0 before:transition-opacity before:duration-(--dur-fast) before:ease-out hover:before:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 max-sm:flex-wrap"
			>
				<span class="flex min-w-0 flex-1 flex-col gap-0.5">
					<span id={titleId} class="text-title">{m.advanced()}</span>
					<span class="text-fg-muted">{summary}</span>
				</span>
				{#if changedCount > 0}
					<span class="max-sm:order-last max-sm:basis-full">
						<Badge>{m.count_changed({ count: changedCount })}</Badge>
					</span>
				{/if}
				<LucideChevronDown
					class="text-fg-muted size-4 shrink-0 transition-transform duration-(--dur-base) ease-in-out group-data-[state=open]/advanced:rotate-180"
				/>
			</Collapsible.Trigger>
			<Collapsible.Content
				class="*:transition-motion flex flex-col gap-10 *:duration-(--dur-base) data-starting-style:*:-translate-y-(--rise) data-starting-style:*:opacity-0"
			>
				{@render children()}
			</Collapsible.Content>
		</section>
	{/snippet}
</Collapsible.Root>

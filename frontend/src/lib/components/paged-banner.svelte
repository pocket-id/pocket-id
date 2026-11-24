<script lang="ts" module>
	// Something that needs attention, such as an account task or a warning about the instance
	export type BannerItem = {
		tone: 'warning' | 'destructive';
		title: string;
		description: string;
		docsHref?: string;
		action?: { label: string; run: () => Promise<void> };
	};
</script>

<script lang="ts">
	import DocsLink from '#lib/components/docs-link.svelte';
	import { Button } from '@pocket-id/ui/button';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '@pocket-id/ui/utils';
	import ChevronLeftIcon from '~icons/solar/alt-arrow-left-linear';
	import ChevronRightIcon from '~icons/solar/alt-arrow-right-linear';
	import TriangleAlertIcon from '~icons/solar/danger-triangle-bold-duotone';

	let { items }: { items: BannerItem[] } = $props();

	// Feeds the tone's colors to the `--tone-*` variables, like the Alert primitive does
	const toneVars: Record<BannerItem['tone'], string> = {
		warning:
			'[--tone-border:var(--warning-border)] [--tone-soft:var(--warning-soft)] [--tone-solid:var(--warning-solid)]',
		destructive:
			'[--tone-border:var(--danger-border)] [--tone-soft:var(--danger-soft)] [--tone-solid:var(--danger-solid)]'
	};

	let index = $state(0);

	// A resolved item leaves the list, so the index is clamped instead of pointing past its end
	const current = $derived(items[Math.min(index, items.length - 1)]);

	function step(by: number) {
		index = (Math.min(index, items.length - 1) + by + items.length) % items.length;
	}
</script>

<!-- One item at a time, most severe first, with the rest behind the pager -->
{#if current}
	<div
		role="status"
		class={cn(
			'text-control text-fg flex flex-wrap items-center gap-x-3 gap-y-2 rounded-md bg-(--tone-soft) px-3 py-2 shadow-[inset_0_0_0_1px_var(--tone-border)]',
			toneVars[current.tone]
		)}
	>
		<!-- The icon sits on the first line of text, while the block as a whole stays centered next to the buttons -->
		<div class="flex min-w-0 flex-1 basis-60 items-start gap-3">
			<TriangleAlertIcon class="mt-0.5 size-4 shrink-0 text-(--tone-solid)" />
			<p class="min-w-0 flex-1">
				<span class="font-medium">{current.title}</span>
				<span class="text-fg-muted">{current.description}</span>
				{#if current.docsHref}
					<DocsLink href={current.docsHref} />
				{/if}
			</p>
		</div>
		<div class="flex items-center gap-1 max-sm:ms-7">
			{#if current.action}
				<Button variant="secondary" size="sm" usePromiseLoading onclick={current.action.run}>
					{current.action.label}
				</Button>
			{/if}
			{#if items.length > 1}
				<Button variant="ghost" size="icon-xs" aria-label={m.previous()} onclick={() => step(-1)}>
					<ChevronLeftIcon />
				</Button>
				<span class="text-caption text-fg-muted tabular-nums">
					{m.step_of_total({ count: items.indexOf(current) + 1, total: items.length })}
				</span>
				<Button variant="ghost" size="icon-xs" aria-label={m.next()} onclick={() => step(1)}>
					<ChevronRightIcon />
				</Button>
			{/if}
		</div>
	</div>
{/if}

<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils/style.ts';
	import { LucideChevronLeft, LucideChevronRight } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	type Crumb = { label: string; href?: string };
	type FactBand = { label: string; items: { label: string; value: string | Snippet }[] };

	let {
		title,
		description,
		breadcrumbs,
		identity,
		status,
		tags,
		actions,
		facts
	}: {
		title: string;
		description?: string | Snippet;
		// The ancestors of a detail page; passing them turns the header into a detail header with the title as the current item
		breadcrumbs?: Crumb[];
		identity?: Snippet;
		status?: Snippet;
		tags?: Snippet;
		actions?: Snippet;
		// Read-only facts that turn a detail header into the sunken identity hero
		facts?: FactBand[];
	} = $props();

	const backCrumb = $derived(breadcrumbs?.findLast((crumb) => crumb.href));
	const hero = $derived(!!facts?.length);
</script>

{#snippet h1()}
	<h1 class="font-display text-display-m text-fg max-sm:text-display-s wrap-anywhere">{title}</h1>
{/snippet}

{#snippet text(value: string | Snippet)}
	{#if typeof value === 'string'}{value}{:else}{@render value()}{/if}
{/snippet}

{#if breadcrumbs}
	<header class={cn(hero && 'sm:bg-surface-sunken sm:rounded-xl sm:px-7 sm:py-6')}>
		<div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2">
			<nav aria-label={m.breadcrumb()} class={cn('min-w-0 sm:col-span-2', hero && 'sm:mb-2')}>
				<ol class="text-caption text-fg-muted flex flex-wrap items-center gap-1 max-sm:hidden">
					{#each breadcrumbs as crumb (crumb.label)}
						<li class="inline-flex items-center gap-1">
							{#if crumb.href}
								<a href={crumb.href} class="hover:text-fg underline-offset-3 hover:underline">
									{crumb.label}
								</a>
							{:else}
								{crumb.label}
							{/if}
							<LucideChevronRight class="size-3" />
						</li>
					{/each}
					<li><span aria-current="page" class="text-fg">{title}</span></li>
				</ol>
				{#if backCrumb}
					<a
						href={backCrumb.href}
						class="text-caption text-fg-muted hover:text-fg inline-flex items-center gap-1 sm:hidden"
					>
						<LucideChevronLeft class="size-3" />
						{backCrumb.label}
					</a>
				{/if}
			</nav>
			<div
				class={cn(
					'col-span-2 flex min-w-0 items-start gap-3 sm:col-span-1',
					hero && 'sm:items-center sm:gap-5'
				)}
			>
				{#if identity}
					<div class={cn('flex shrink-0 *:size-10', hero && 'sm:*:size-14')}>
						{@render identity()}
					</div>
				{/if}
				<div class="min-w-0 flex-1">
					<div class="flex flex-wrap items-center gap-x-3 gap-y-1">
						{@render h1()}
						{@render status?.()}
					</div>
					{#if tags}
						<div class="text-control text-fg-muted mt-2 flex flex-wrap items-center gap-2">
							{@render tags()}
						</div>
					{/if}
				</div>
			</div>
			{#if actions}
				<div
					class={cn(
						'col-start-2 row-start-1 flex items-center gap-2 justify-self-end sm:row-start-2',
						!hero && 'sm:self-start'
					)}
				>
					{@render actions()}
				</div>
			{/if}
		</div>
		{#if facts?.length}
			<!-- Narrower windows put each band's label above two columns, so IDs and paths are not cut to a few characters -->
			<div class="mt-6 flex flex-col gap-5">
				{#each facts as band (band.label)}
					<div
						class="grid gap-x-6 gap-y-3 pt-5 shadow-[inset_0_1px_0_var(--border)] xl:grid-cols-[120px_minmax(0,1fr)]"
					>
						<p class="text-caption text-fg-muted font-semibold">{band.label}</p>
						<dl
							class={cn(
								'grid gap-x-6 gap-y-3 sm:grid-cols-2',
								band.items.length > 2 && 'xl:grid-cols-3'
							)}
						>
							{#each band.items as item (item.label)}
								<div class="min-w-0">
									<dt class="text-caption text-fg-muted">{item.label}</dt>
									<dd class="text-control text-fg">{@render text(item.value)}</dd>
								</div>
							{/each}
						</dl>
					</div>
				{/each}
			</div>
		{/if}
	</header>
{:else}
	<!-- Actions that leave the title too little room wrap under it instead of squeezing it -->
	<header class="flex flex-wrap items-start gap-4">
		<div class="flex min-w-40 flex-1 flex-col gap-2">
			{@render h1()}
			{#if description}
				<p class="text-body text-fg-muted">{@render text(description)}</p>
			{/if}
		</div>
		{#if actions}
			<!-- The primary action stays on the title row and drops to the small button size on phones -->
			<div
				class="flex shrink-0 items-center gap-2 max-sm:*:data-[slot=button]:h-(--h-sm) max-sm:*:data-[slot=button]:px-3"
			>
				{@render actions()}
			</div>
		{/if}
	</header>
{/if}

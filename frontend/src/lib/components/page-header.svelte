<script lang="ts">
	import { m } from '#lib/paraglide/messages.js';
	import ChevronLeftIcon from '~icons/solar/alt-arrow-left-linear';
	import ChevronRightIcon from '~icons/solar/alt-arrow-right-linear';
	import type { Snippet } from 'svelte';

	type Crumb = { label: string; href?: string };

	let {
		title,
		description,
		breadcrumbs,
		identity,
		status,
		tags,
		actions
	}: {
		title: string;
		description?: string | Snippet;
		// The ancestors of a detail page; passing them turns the header into a detail header with the title as the current item
		breadcrumbs?: Crumb[];
		identity?: Snippet;
		status?: Snippet;
		tags?: Snippet;
		actions?: Snippet;
	} = $props();

	const backCrumb = $derived(breadcrumbs?.findLast((crumb) => crumb.href));
</script>

{#snippet h1()}
	<h1 class="font-display text-display-m text-fg max-sm:text-display-s wrap-anywhere">{title}</h1>
{/snippet}

{#snippet text(value: string | Snippet)}
	{#if typeof value === 'string'}{value}{:else}{@render value()}{/if}
{/snippet}

{#if breadcrumbs}
	<header>
		<div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2">
			<nav aria-label={m.breadcrumb()} class="min-w-0 sm:col-span-2">
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
							<ChevronRightIcon class="size-3" />
						</li>
					{/each}
					<li><span aria-current="page" class="text-fg">{title}</span></li>
				</ol>
				{#if backCrumb}
					<a
						href={backCrumb.href}
						class="text-caption text-fg-muted hover:text-fg inline-flex items-center gap-1 sm:hidden"
					>
						<ChevronLeftIcon class="size-3" />
						{backCrumb.label}
					</a>
				{/if}
			</nav>
			<div class="col-span-2 flex min-w-0 items-start gap-3 sm:col-span-1">
				{#if identity}
					<div class="flex shrink-0 *:size-10">
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
					{#if description}
						<p class="text-body text-fg-muted mt-2">{@render text(description)}</p>
					{/if}
				</div>
			</div>
			{#if actions}
				<div
					class="col-start-2 row-start-1 flex items-center gap-2 justify-self-end sm:row-start-2 sm:self-start"
				>
					{@render actions()}
				</div>
			{/if}
		</div>
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

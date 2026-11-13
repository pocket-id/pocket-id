<script lang="ts">
	import EnvOverrideLock from '#lib/components/env-override-lock.svelte';
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		action,
		envOverride = false,
		children
	}: {
		title: string;
		description?: string | Snippet;
		action?: Snippet;
		// Marks the setting the card consists of as set by an environment variable
		envOverride?: boolean;
		// Blocks inside bring their own vertical padding, like setting rows do, so fields and rows can share a card
		children: Snippet;
	} = $props();

	const headingId = $props.id();
</script>

<!-- Groups of settings are cards, while a list that fills a tab stays open in a settings section -->
<section aria-labelledby={headingId} class="rounded-xl border">
	<!-- The band's corners follow the inside of the card's border, and on phones an action moves below the text instead of squeezing it -->
	<div
		class="bg-surface-sunken flex min-h-13 items-center gap-x-4 gap-y-2 rounded-t-[calc(var(--radius-xl)-1px)] border-b px-4 py-2 max-sm:flex-col max-sm:items-start max-sm:justify-center sm:px-5"
	>
		<div class="flex min-w-0 flex-1 flex-col">
			<div class="flex items-center gap-1">
				<h2 id={headingId} class="text-heading text-fg">{title}</h2>
				{#if envOverride}
					<EnvOverrideLock />
				{/if}
			</div>
			{#if description}
				<p class="text-caption text-fg-muted">
					{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
				</p>
			{/if}
		</div>
		{#if action}
			<div class="flex shrink-0 items-center gap-2 empty:hidden">{@render action()}</div>
		{/if}
	</div>
	<div class="px-4 py-1 sm:px-5">
		{@render children()}
	</div>
</section>

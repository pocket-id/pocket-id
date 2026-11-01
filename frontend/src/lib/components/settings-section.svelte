<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		action,
		children
	}: {
		title: string;
		description?: string | Snippet;
		action?: Snippet;
		children: Snippet;
	} = $props();

	const headingId = $props.id();
</script>

<section aria-labelledby={headingId} class="flex flex-col">
	<div class="flex items-end gap-4 pb-2 max-sm:flex-col max-sm:items-start">
		<div class="flex min-w-0 flex-1 flex-col gap-1">
			<h2 id={headingId} class="text-title text-fg">{title}</h2>
			{#if description}
				<p class="text-caption text-fg-muted max-w-[68ch]">
					{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
				</p>
			{/if}
		</div>
		{#if action}
			<div class="flex shrink-0 items-center gap-2 empty:hidden">{@render action()}</div>
		{/if}
	</div>
	{@render children()}
</section>

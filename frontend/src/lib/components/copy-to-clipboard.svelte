<script lang="ts">
	import * as Tooltip from '#lib/components/ui/tooltip/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { LucideCheck } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let { value, children }: { value: string; children: Snippet } = $props();

	let open = $state(false);
	let copied = $state(false);

	function onOpenChange(state: boolean) {
		open = state;
		if (!state) {
			setTimeout(() => (copied = false), 500);
		}
	}

	async function copyToClipboard() {
		open = true;
		await navigator.clipboard.writeText(value);
		copied = true;
		setTimeout(() => onOpenChange(false), 1500);
	}
</script>

<Tooltip.Provider>
	<Tooltip.Root {onOpenChange} {open} disableCloseOnTriggerClick>
		<Tooltip.Trigger
			class="hover:bg-fill-hover focus-visible:outline-ring -mx-1 cursor-copy rounded-xs px-1 text-start transition-colors duration-(--dur-fast) focus-visible:outline-2 focus-visible:outline-offset-2"
			onclick={copyToClipboard}
		>
			{@render children()}
		</Tooltip.Trigger>
		<Tooltip.Content>
			{#if copied}
				<LucideCheck class="size-3.5" />
				{m.copied()}
			{:else}
				{m.copy()}
			{/if}
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>
<span class="sr-only" aria-live="polite">{copied ? m.copied() : ''}</span>

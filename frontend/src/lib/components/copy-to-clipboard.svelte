<script lang="ts">
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { m } from '$lib/paraglide/messages';
	import { LucideCheck } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let { value, children }: { value: string; children: Snippet } = $props();

	let open = $state(false);
	let copied = $state(false);

	function onClick() {
		open = true;
		copyToClipboard();
	}

	function onOpenChange(state: boolean) {
		open = state;
		if (!state) {
			setTimeout(() => (copied = false), 500);
		}
	}

	function copyToClipboard() {
		navigator.clipboard.writeText(value);
		copied = true;
		setTimeout(() => onOpenChange(false), 1000);
	}
</script>

<Tooltip.Provider>
	<Tooltip.Root {onOpenChange} {open} disableCloseOnTriggerClick>
		<Tooltip.Trigger class="text-start" onclick={onClick}>{@render children()}</Tooltip.Trigger>
		<Tooltip.Content onclick={copyToClipboard}>
			{#if copied}
				<span class="flex items-center"><LucideCheck class="mr-1 size-4" /> {m.copied()}</span>
			{:else}
				<span>{m.click_to_copy()}</span>
			{/if}
		</Tooltip.Content>
	</Tooltip.Root>
</Tooltip.Provider>

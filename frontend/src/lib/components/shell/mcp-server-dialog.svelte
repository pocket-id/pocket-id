<script lang="ts">
	import { page } from '$app/state';
	import CopyButton from '#lib/components/copy-button.svelte';
	import DocsLink from '#lib/components/docs-link.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { Button } from '@pocket-id/ui/button';
	import * as Dialog from '@pocket-id/ui/dialog';

	let { open = $bindable() }: { open: boolean } = $props();

	const mcpServerUrl = `${page.url.origin}/mcp`;
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-(--dialog-md)">
		<Dialog.Header>
			<Dialog.Title>{m.mcp_server()}</Dialog.Title>
			<Dialog.Description>
				{m.mcp_server_description()}
				<DocsLink href="https://pocket-id.org/docs/guides/mcp" />
			</Dialog.Description>
		</Dialog.Header>
		<div
			class="text-control bg-surface-sunken flex min-h-9 items-center gap-2 rounded-sm px-3 py-1"
		>
			<span class="text-mono-s min-w-0 flex-1 truncate font-mono" title={mcpServerUrl}
				>{mcpServerUrl}</span
			>
			<CopyButton value={mcpServerUrl} label={m.copy_name({ name: m.mcp_server_url() })} />
		</div>
		<Dialog.Footer>
			<Button variant="secondary" onclick={() => (open = false)}>{m.close()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

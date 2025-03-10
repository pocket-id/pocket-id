<script lang="ts">
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import type { ApiKeyResponse } from '$lib/types/api-key.type';

	let {
		apiKeyResponse = $bindable(),
		onOpenChange
	}: {
		apiKeyResponse: ApiKeyResponse | null;
		onOpenChange: (open: boolean) => void;
	} = $props();
</script>

<Dialog.Root open={!!apiKeyResponse} {onOpenChange}>
	<Dialog.Content class="max-w-md" closeButton={false}>
		<Dialog.Header>
			<Dialog.Title>API Key Created</Dialog.Title>
			<Dialog.Description>
				For security reasons, this key will only be shown once. Please store it securely.
			</Dialog.Description>
		</Dialog.Header>
		{#if apiKeyResponse}
			<div>
				<div class="mb-2 font-medium">Name</div>
				<p class="text-muted-foreground">{apiKeyResponse.apiKey.name}</p>

				{#if apiKeyResponse.apiKey.description}
					<div class="mb-2 mt-4 font-medium">Description</div>
					<p class="text-muted-foreground">{apiKeyResponse.apiKey.description}</p>
				{/if}

				<div class="mb-2 mt-4 font-medium">API Key</div>
				<div class="bg-muted rounded-md p-2">
					<CopyToClipboard value={apiKeyResponse.token}>
						<span class="break-all font-mono text-sm">{apiKeyResponse.token}</span>
					</CopyToClipboard>
				</div>
			</div>
		{/if}
		<Dialog.Footer class="mt-3">
			<Button variant="default" on:click={() => onOpenChange(false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

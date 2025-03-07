<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
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
	<Dialog.Content class="max-w-md">
		<Dialog.Header>
			<Dialog.Title>API Key Created</Dialog.Title>
			<Dialog.Description>
				Copy your API key now. For security reasons, it won't be displayed again.
			</Dialog.Description>
		</Dialog.Header>
		{#if apiKeyResponse}
			<div class="my-4">
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

				<div class="mt-4 text-sm text-red-500">
					Important: This key will only be shown once. Store it securely.
				</div>
			</div>
		{/if}
		<Dialog.Footer>
			<Button variant="secondary" on:click={() => onOpenChange(false)}>Close</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

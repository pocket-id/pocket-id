<script lang="ts">
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import type { ApiKeyResponse } from '$lib/types/api-key.type';

	let {
		apiKeyResponse = $bindable()
	}: {
		apiKeyResponse: ApiKeyResponse | null;
	} = $props();

	function onOpenChange(open: boolean) {
		if (!open) {
			apiKeyResponse = null;
		}
	}
</script>

<Dialog.Root open={!!apiKeyResponse} {onOpenChange}>
	<Dialog.Content class="max-w-md" onOpenAutoFocus={(e) => e.preventDefault()}>
		<Dialog.Header>
			<Dialog.Title>{m.api_key_created()}</Dialog.Title>
			<Dialog.Description>
				{m.for_security_reasons_this_key_will_only_be_shown_once()}
			</Dialog.Description>
		</Dialog.Header>
		{#if apiKeyResponse}
			<div>
				<div class="mb-2 font-medium">{m.name()}</div>
				<p class="text-muted-foreground">{apiKeyResponse.apiKey.name}</p>
				{#if apiKeyResponse.apiKey.description}
					<div class="mt-4 mb-2 font-medium">{m.description()}</div>
					<p class="text-muted-foreground">{apiKeyResponse.apiKey.description}</p>
				{/if}

				<div class="mt-4 mb-2 font-medium">{m.api_key()}</div>
				<div class="bg-muted rounded-md p-2">
					<CopyToClipboard value={apiKeyResponse.token}>
						<span class="font-mono text-sm break-all">{apiKeyResponse.token}</span>
					</CopyToClipboard>
				</div>
			</div>
		{/if}
		<Dialog.Footer class="mt-3">
			<Button variant="default" onclick={() => onOpenChange(false)}>{m.close()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

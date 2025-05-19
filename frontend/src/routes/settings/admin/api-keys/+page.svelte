<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, ShieldEllipsis, ShieldPlus } from '@lucide/svelte';
	import { slide } from 'svelte/transition';
	import ApiKeyDialog from './api-key-dialog.svelte';
	import ApiKeyForm from './api-key-form.svelte';
	import ApiKeyList from './api-key-list.svelte';

	let { data } = $props();
	let apiKeys = $state(data.apiKeys);
	let apiKeysRequestOptions = $state(data.apiKeysRequestOptions);

	const apiKeyService = new ApiKeyService();
	let expandAddApiKey = $state(false);
	let apiKeyResponse = $state<ApiKeyResponse | null>(null);

	async function createApiKey(apiKeyData: ApiKeyCreate) {
		try {
			const response = await apiKeyService.create(apiKeyData);
			apiKeyResponse = response;

			// After creation, reload the list of API keys
			apiKeys = await apiKeyService.list(apiKeysRequestOptions);

			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}
</script>

<svelte:head>
	<title>{m.api_keys()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<div>
					<Card.Title>
						<ShieldPlus class="text-primary/80 h-5 w-5" />
						{m.create_api_key()}
					</Card.Title>
					<Card.Description>{m.add_a_new_api_key_for_programmatic_access()}</Card.Description>
				</div>
				{#if !expandAddApiKey}
					<Button onclick={() => (expandAddApiKey = true)}>{m.add_api_key()}</Button>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddApiKey = false)}>
						<LucideMinus class="h-5 w-5" />
					</Button>
				{/if}
			</div>
		</Card.Header>
		{#if expandAddApiKey}
			<div transition:slide>
				<Card.Content>
					<ApiKeyForm callback={createApiKey} />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<ShieldEllipsis class="text-primary/80 h-5 w-5" />
				{m.manage_api_keys()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<ApiKeyList {apiKeys} requestOptions={apiKeysRequestOptions} />
		</Card.Content>
	</Card.Root>
</div>

<ApiKeyDialog bind:apiKeyResponse />

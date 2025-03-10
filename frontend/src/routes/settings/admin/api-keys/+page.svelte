<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKey, ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
	import type { Paginated } from '$lib/types/pagination.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus } from 'lucide-svelte';
	import { slide } from 'svelte/transition';
	import ApiKeyDialog from './api-key-dialog.svelte';
	import ApiKeyForm from './api-key-form.svelte';
	import ApiKeyList from './api-key-list.svelte';

	let { data } = $props();
	let apiKeys = $state<Paginated<ApiKey>>(data);

	const apiKeyService = new ApiKeyService();
	let expandAddApiKey = $state(false);
	let apiKeyResponse = $state<ApiKeyResponse | null>(null);

	async function createApiKey(apiKeyData: ApiKeyCreate) {
		try {
			const response = await apiKeyService.create(apiKeyData);
			apiKeyResponse = response;

			// After creation, reload the list of API keys
			const updatedKeys = await apiKeyService.list();
			apiKeys = updatedKeys;

			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	function handleDialogClose(open: boolean) {
		if (!open) {
			apiKeyResponse = null;
			// Refresh the list when dialog closes
			apiKeyService
				.list()
				.then((keys) => {
					apiKeys = keys;
				})
				.catch(axiosErrorToast);
		}
	}
</script>

<svelte:head>
	<title>API Keys</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>Create API Key</Card.Title>
				<Card.Description>Add a new API key for programmatic access.</Card.Description>
			</div>
			{#if !expandAddApiKey}
				<Button on:click={() => (expandAddApiKey = true)}>Add API Key</Button>
			{:else}
				<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddApiKey = false)}>
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

<Card.Root class="mt-6">
	<Card.Header>
		<Card.Title>Manage API Keys</Card.Title>
	</Card.Header>
	<Card.Content>
		<ApiKeyList {apiKeys} />
	</Card.Content>
</Card.Root>

<ApiKeyDialog bind:apiKeyResponse onOpenChange={handleDialogClose} />

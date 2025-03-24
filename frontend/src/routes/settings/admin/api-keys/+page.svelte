<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import ApiKeyService from '$lib/services/api-key-service';
	import type { ApiKeyCreate, ApiKeyResponse } from '$lib/types/api-key.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, ShieldEllipsis, ShieldPlus } from 'lucide-svelte';
	import { slide } from 'svelte/transition';
	import { onMount } from 'svelte';
	import ApiKeyDialog from './api-key-dialog.svelte';
	import ApiKeyForm from './api-key-form.svelte';
	import ApiKeyList from './api-key-list.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let apiKeys = $state(data.apiKeys);
	let apiKeysRequestOptions = $state(data.apiKeysRequestOptions);

	const apiKeyService = new ApiKeyService();
	let expandAddApiKey = $state(false);
	let apiKeyResponse = $state<ApiKeyResponse | null>(null);
	let mounted = $state(false);

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

	onMount(() => {
		mounted = true;
	});
</script>

<svelte:head>
	<title>{m.api_keys()}</title>
</svelte:head>

{#if mounted}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<div class="flex items-center justify-between">
					<div>
						<Card.Title class="flex items-center gap-2 text-xl font-semibold">
							<ShieldPlus class="text-primary/80 h-5 w-5" />
							{m.create_api_key()}
						</Card.Title>
						<Card.Description>{m.add_a_new_api_key_for_programmatic_access()}</Card.Description>
					</div>
					{#if !expandAddApiKey}
						<Button on:click={() => (expandAddApiKey = true)}>{m.add_api_key()}</Button>
					{:else}
						<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddApiKey = false)}>
							<LucideMinus class="h-5 w-5" />
						</Button>
					{/if}
				</div>
			</Card.Header>
			{#if expandAddApiKey}
				<div transition:slide>
					<Card.Content class="bg-muted/20 pt-5">
						<ApiKeyForm callback={createApiKey} />
					</Card.Content>
				</div>
			{/if}
		</Card.Root>
	</div>

	<div class="animate-fade-in" style="animation-delay: 200ms;">
		<Card.Root class="mt-6">
			<Card.Header class="border-b">
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					<ShieldEllipsis class="text-primary/80 h-5 w-5" />
					{m.manage_api_keys()}
				</Card.Title>
			</Card.Header>
			<Card.Content class="bg-muted/20 pt-5">
				<ApiKeyList {apiKeys} requestOptions={apiKeysRequestOptions} />
			</Card.Content>
		</Card.Root>
	</div>
{/if}

<ApiKeyDialog bind:apiKeyResponse />

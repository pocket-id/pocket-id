<script lang="ts">
	import FormattedMessage from '$lib/components/formatted-message.svelte';
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

	const apiKeyService = new ApiKeyService();
	let expandAddApiKey = $state(false);
	let apiKeyResponse = $state<ApiKeyResponse | null>(null);
	let listRef: ApiKeyList;

	async function createApiKey(apiKeyData: ApiKeyCreate) {
		try {
			const response = await apiKeyService.create(apiKeyData);
			apiKeyResponse = response;
			listRef.refresh();
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

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>
					<ShieldPlus class="text-primary/80 size-5" />
					{m.create_api_key()}
				</Card.Title>
				<Card.Description
					><FormattedMessage m={m.add_a_new_api_key_for_programmatic_access()} /></Card.Description
				>
			</div>
			{#if !expandAddApiKey}
				<Button onclick={() => (expandAddApiKey = true)}>{m.add_api_key()}</Button>
			{:else}
				<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddApiKey = false)}>
					<LucideMinus class="size-5" />
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

<Card.Root class="gap-0">
	<Card.Header>
		<Card.Title>
			<ShieldEllipsis class="text-primary/80 size-5" />
			{m.manage_api_keys()}
		</Card.Title>
	</Card.Header>
	<Card.Content>
		<ApiKeyList bind:this={listRef} />
	</Card.Content>
</Card.Root>

<ApiKeyDialog title={m.api_key_created()} bind:apiKeyResponse />

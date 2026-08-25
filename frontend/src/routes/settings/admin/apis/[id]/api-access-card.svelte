<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Tabs from '$lib/components/ui/tabs';
	import { m } from '$lib/paraglide/messages';
	import type { Api, ApiCimdAccessUpdate } from '$lib/types/api.type';
	import ApiCimdAccessTab from './api-cimd-access-tab.svelte';
	import ApiClientsTab from './api-clients-tab.svelte';

	let {
		api,
		onCimdAccessSave
	}: { api: Api; onCimdAccessSave: (update: ApiCimdAccessUpdate) => Promise<void> } = $props();

	let clientsTab = $state<ApiClientsTab>();
	let tab = $state('clients');

	export async function refresh() {
		await clientsTab?.refresh();
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>{m.access()}</Card.Title>
		<Card.Description>{m.api_access_card_description()}</Card.Description>
	</Card.Header>
	<Card.Content>
		<Tabs.Root bind:value={tab} class="gap-4">
			<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
				<Tabs.List>
					<Tabs.Trigger value="clients">{m.oidc_clients()}</Tabs.Trigger>
					<Tabs.Trigger value="metadata-document-clients">
						{m.metadata_document_client_access()}
					</Tabs.Trigger>
				</Tabs.List>
				{#if tab === 'clients'}
					<Button variant="outline" onclick={() => clientsTab?.openPicker()}>
						{m.add_client()}
					</Button>
				{/if}
			</div>

			<Tabs.Content value="clients">
				<ApiClientsTab bind:this={clientsTab} {api} />
			</Tabs.Content>
			<Tabs.Content value="metadata-document-clients">
				<ApiCimdAccessTab {api} onSave={onCimdAccessSave} />
			</Tabs.Content>
		</Tabs.Root>
	</Card.Content>
</Card.Root>

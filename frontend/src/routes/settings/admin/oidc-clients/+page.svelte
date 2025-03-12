<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import OIDCService from '$lib/services/oidc-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import OIDCClientForm from './oidc-client-form.svelte';
	import OIDCClientList from './oidc-client-list.svelte';

	let { data } = $props();
	let clients = $state(data.clients);
	let clientsRequestOptions = $state(data.clientsRequestOptions);
	let expandAddClient = $state(false);

	const oidcService = new OIDCService();

	async function createOIDCClient(client: OidcClientCreateWithLogo) {
		try {
			const createdClient = await oidcService.createClient(client);
			if (client.logo) {
				await oidcService.updateClientLogo(createdClient, client.logo);
			}
			const clientSecret = await oidcService.createClientSecret(createdClient.id);
			clientSecretStore.set(clientSecret);
			goto(`/settings/admin/oidc-clients/${createdClient.id}`);
			toast.success('OIDC client created successfully');
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}
</script>

<svelte:head>
	<title>OIDC Clients</title>
</svelte:head>

<Card.Root>
	<Card.Header>
		<div class="flex items-center justify-between">
			<div>
				<Card.Title>Create OIDC Client</Card.Title>
				<Card.Description>Add a new OIDC client to {$appConfigStore.appName}.</Card.Description>
			</div>
			{#if !expandAddClient}
				<Button on:click={() => (expandAddClient = true)}>Add OIDC Client</Button>
			{:else}
				<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddClient = false)}>
					<LucideMinus class="h-5 w-5" />
				</Button>
			{/if}
		</div>
	</Card.Header>
	{#if expandAddClient}
		<div transition:slide>
			<Card.Content>
				<OIDCClientForm callback={createOIDCClient} />
			</Card.Content>
		</div>
	{/if}
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Manage OIDC Clients</Card.Title>
	</Card.Header>
	<Card.Content>
		<OIDCClientList {clients} requestOptions={clientsRequestOptions} />
	</Card.Content>
</Card.Root>

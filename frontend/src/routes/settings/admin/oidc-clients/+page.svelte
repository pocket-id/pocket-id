<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import OIDCService from '$lib/services/oidc-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, ShieldCheck, ShieldPlus } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import OIDCClientForm from './oidc-client-form.svelte';
	import OIDCClientList from './oidc-client-list.svelte';
	import { m } from '$lib/paraglide/messages';
	import { onMount } from 'svelte';

	let { data } = $props();
	let clients = $state(data.clients);
	let clientsRequestOptions = $state(data.clientsRequestOptions);
	let expandAddClient = $state(false);
	let mounted = $state(false);

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
			toast.success(m.oidc_client_created_successfully());
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
	<title>{m.oidc_clients()}</title>
</svelte:head>

{#if mounted}
	<div class="animate-fade-in" style="animation-delay: 100ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<div class="flex items-center justify-between">
					<div>
						<Card.Title class="flex items-center gap-2 text-xl font-semibold">
							<ShieldPlus class="text-primary/80 h-5 w-5" />
							{m.create_oidc_client()}
						</Card.Title>
						<Card.Description
							>{m.add_a_new_oidc_client_to_appname({
								appName: $appConfigStore.appName
							})}</Card.Description
						>
					</div>
					{#if !expandAddClient}
						<Button on:click={() => (expandAddClient = true)}>{m.add_oidc_client()}</Button>
					{:else}
						<Button class="h-8 p-3" variant="ghost" on:click={() => (expandAddClient = false)}>
							<LucideMinus class="h-5 w-5" />
						</Button>
					{/if}
				</div>
			</Card.Header>
			{#if expandAddClient}
				<div transition:slide>
					<Card.Content class="bg-muted/20 pt-5">
						<OIDCClientForm callback={createOIDCClient} />
					</Card.Content>
				</div>
			{/if}
		</Card.Root>
	</div>

	<div class="animate-fade-in" style="animation-delay: 200ms;">
		<Card.Root>
			<Card.Header class="border-b">
				<Card.Title class="flex items-center gap-2 text-xl font-semibold">
					<ShieldCheck class="text-primary/80 h-5 w-5" />
					{m.manage_oidc_clients()}
				</Card.Title>
			</Card.Header>
			<Card.Content class="bg-muted/20 pt-5">
				<OIDCClientList {clients} requestOptions={clientsRequestOptions} />
			</Card.Content>
		</Card.Root>
	</div>
{/if}

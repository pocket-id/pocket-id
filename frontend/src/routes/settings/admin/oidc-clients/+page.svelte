<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { m } from '$lib/paraglide/messages';
	import OIDCService from '$lib/services/oidc-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideMinus, ShieldCheck, ShieldPlus } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import OIDCClientForm from './oidc-client-form.svelte';
	import OIDCClientList from './oidc-client-list.svelte';

	let expandAddClient = $state(false);

	const oidcService = new OIDCService();

	async function createOIDCClient(client: OidcClientCreateWithLogo) {
		try {
			const createdClient = await oidcService.createClient(client);

			const logoPromise = client.logo
				? oidcService.updateClientLogo(createdClient, client.logo, true)
				: Promise.resolve();
			const darkLogoPromise = client.darkLogo
				? oidcService.updateClientLogo(createdClient, client.darkLogo, false)
				: Promise.resolve();
			await Promise.all([logoPromise, darkLogoPromise]);

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
</script>

<svelte:head>
	<title>{m.oidc_clients()}</title>
</svelte:head>

<div>
	<Card.Root>
		<Card.Header>
			<div class="flex items-center justify-between">
				<div>
					<Card.Title>
						<ShieldPlus class="text-primary/80 size-5" />
						{m.create_oidc_client()}
					</Card.Title>
					<Card.Description
						>{m.add_a_new_oidc_client_to_appname({
							appName: $appConfigStore.appName
						})}</Card.Description
					>
				</div>
				{#if !expandAddClient}
					<Button onclick={() => (expandAddClient = true)}>{m.add_oidc_client()}</Button>
				{:else}
					<Button class="h-8 p-3" variant="ghost" onclick={() => (expandAddClient = false)}>
						<LucideMinus class="size-5" />
					</Button>
				{/if}
			</div>
		</Card.Header>
		{#if expandAddClient}
			<div transition:slide>
				<Card.Content>
					<OIDCClientForm mode="create" callback={createOIDCClient} />
				</Card.Content>
			</div>
		{/if}
	</Card.Root>
</div>

<div>
	<Card.Root>
		<Card.Header>
			<Card.Title>
				<ShieldCheck class="text-primary/80 size-5" />
				{m.manage_oidc_clients()}
			</Card.Title>
		</Card.Header>
		<Card.Content>
			<OIDCClientList />
		</Card.Content>
	</Card.Root>
</div>

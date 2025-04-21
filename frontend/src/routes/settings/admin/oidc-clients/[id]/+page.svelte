<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import Label from '$lib/components/ui/label/label.svelte';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import OidcService from '$lib/services/oidc-service';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft, LucideRefreshCcw } from 'lucide-svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import OidcForm from '../oidc-client-form.svelte';
	import { m } from '$lib/paraglide/messages';

	let { data } = $props();
	let client = $state({
		...data,
		allowedUserGroupIds: data.allowedUserGroups.map((g) => g.id)
	});
	let showAllDetails = $state(false);

	const oidcService = new OidcService();

	const setupDetails = $state({
		[m.authorization_url()]: `https://${$page.url.hostname}/authorize`,
		[m.oidc_discovery_url()]: `https://${$page.url.hostname}/.well-known/openid-configuration`,
		[m.token_url()]: `https://${$page.url.hostname}/api/oidc/token`,
		[m.userinfo_url()]: `https://${$page.url.hostname}/api/oidc/userinfo`,
		[m.logout_url()]: `https://${$page.url.hostname}/api/oidc/end-session`,
		[m.certificate_url()]: `https://${$page.url.hostname}/.well-known/jwks.json`,
		[m.pkce()]: client.pkceEnabled ? m.enabled() : m.disabled()
	});

	async function updateClient(updatedClient: OidcClientCreateWithLogo) {
		let success = true;
		const dataPromise = oidcService.updateClient(client.id, updatedClient);
		const imagePromise =
			updatedClient.logo !== undefined
				? oidcService.updateClientLogo(client, updatedClient.logo)
				: Promise.resolve();

		client.isPublic = updatedClient.isPublic;
		setupDetails[m.pkce()] = updatedClient.pkceEnabled ? m.enabled() : m.disabled();

		await Promise.all([dataPromise, imagePromise])
			.then(() => {
				toast.success(m.oidc_client_updated_successfully());
			})
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
	}

	async function createClientSecret() {
		openConfirmDialog({
			title: m.create_new_client_secret(),
			message:
				m.are_you_sure_you_want_to_create_a_new_client_secret(),
			confirm: {
				label: m.generate(),
				destructive: true,
				action: async () => {
					try {
						const clientSecret = await oidcService.createClientSecret(client.id);
						clientSecretStore.set(clientSecret);
						toast.success(m.new_client_secret_created_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}

	async function updateUserGroupClients(allowedGroups: string[]) {
		await oidcService
			.updateAllowedUserGroups(client.id, allowedGroups)
			.then(() => {
				toast.success(m.allowed_user_groups_updated_successfully());
			})
			.catch((e) => {
				axiosErrorToast(e);
			});
	}

	beforeNavigate(() => {
		clientSecretStore.clear();
	});
</script>

<svelte:head>
	<title>{m.oidc_client_name({ name: client.name })}</title>
</svelte:head>

<div>
	<a class="text-muted-foreground flex text-sm" href="/settings/admin/oidc-clients"
		><LucideChevronLeft class="h-5 w-5" /> {m.back()}</a
	>
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>{client.name}</Card.Title>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-col">
			<div class="mb-2 flex flex-col sm:flex-row sm:items-center">
				<Label class="mb-0 w-44">{m.client_id()}</Label>
				<CopyToClipboard value={client.id}>
					<span class="text-muted-foreground text-sm" data-testid="client-id"> {client.id}</span>
				</CopyToClipboard>
			</div>
			{#if !client.isPublic}
				<div class="mb-2 mt-1 flex flex-col sm:flex-row sm:items-center">
					<Label class="mb-0 w-44">{m.client_secret()}</Label>
					{#if $clientSecretStore}
						<CopyToClipboard value={$clientSecretStore}>
							<span class="text-muted-foreground text-sm" data-testid="client-secret">
								{$clientSecretStore}
							</span>
						</CopyToClipboard>
					{:else}
						<div>
							<span class="text-muted-foreground text-sm" data-testid="client-secret"
								>••••••••••••••••••••••••••••••••</span
							>
							<Button
								class="ml-2"
								onclick={createClientSecret}
								size="sm"
								variant="ghost"
								aria-label="Create new client secret"><LucideRefreshCcw class="h-3 w-3" /></Button
							>
						</div>
					{/if}
				</div>
			{/if}
			{#if showAllDetails}
				<div transition:slide>
					{#each Object.entries(setupDetails) as [key, value]}
						<div class="mb-5 flex flex-col sm:flex-row sm:items-center">
							<Label class="mb-0 w-44">{key}</Label>
							<CopyToClipboard {value}>
								<span class="text-muted-foreground text-sm">{value}</span>
							</CopyToClipboard>
						</div>
					{/each}
				</div>
			{/if}

			{#if !showAllDetails}
				<div class="mt-4 flex justify-center">
					<Button on:click={() => (showAllDetails = true)} size="sm" variant="ghost"
						>{m.show_more_details()}</Button
					>
				</div>
			{/if}
		</div>
	</Card.Content>
</Card.Root>
<Card.Root>
	<Card.Content class="p-5">
		<OidcForm existingClient={client} callback={updateClient} />
	</Card.Content>
</Card.Root>
<CollapsibleCard
	id="allowed-user-groups"
	title={m.allowed_user_groups()}
	description={m.add_user_groups_to_this_client_to_restrict_access_to_users_in_these_groups()}
>
	<UserGroupSelection bind:selectedGroupIds={client.allowedUserGroupIds} />
	<div class="mt-5 flex justify-end">
		<Button on:click={() => updateUserGroupClients(client.allowedUserGroupIds)}>{m.save()}</Button>
	</div>
</CollapsibleCard>

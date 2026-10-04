<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import FormattedMessage from '#lib/components/formatted-message.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Card from '#lib/components/ui/card/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import ScimService from '#lib/services/scim-service.ts';
	import clientSecretStore from '#lib/stores/client-secret-store.ts';
	import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';
	import type {
		OidcClientCreateWithLogo,
		OidcClientCredentials,
		OidcClientFederatedIdentity,
		OidcClientSecret,
		OidcClientTokenLifetimes
	} from '#lib/types/oidc.type.ts';
	import type { ScimServiceProviderCreate } from '#lib/types/scim.type.ts';
	import { cachedOidcClientLogo } from '#lib/utils/cached-image-util.ts';
	import { LucideChevronLeft, LucideEye, LucideInfo } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { backNavigate } from '../../users/navigate-back-util';
	import { deleteClientAction, refreshClientAction } from '../oidc-client-actions';
	import OidcForm from '../oidc-client-form.svelte';
	import OidcClientPreviewModal from '../oidc-client-preview-modal.svelte';
	import ApiAccessCard from './api-access-card.svelte';
	import OidcClientAllowedUserGroupsCard from './oidc-client-allowed-user-groups-card.svelte';
	import OidcClientConnectionDetailsCard from './oidc-client-connection-details-card.svelte';
	import OidcClientFederatedCredentialsCard from './oidc-client-federated-credentials-card.svelte';
	import OidcClientSecretsCard from './oidc-client-secrets-card.svelte';
	import OidcClientTokenLifetimesCard from './oidc-client-token-lifetimes-card.svelte';
	import ScimResourceProviderForm from './scim-resource-provider-form.svelte';

	let { data } = $props();
	let client = $state({
		...data.client,
		allowedUserGroupIds: data.client.allowedUserGroups.map((g) => g.id)
	});
	// Secrets are managed by their own endpoints, so they are kept out of the client object that the forms below submit
	let clientSecrets = $state<OidcClientSecret[]>(data.client.credentials?.secrets ?? []);

	let scimServiceProvider = $state(data.scimServiceProvider);
	let showPreview = $state(false);
	// Bumped after the client was reloaded so the forms, which only read the client on mount, pick up the new values
	let reloadCount = $state(0);

	const credentialCount = $derived(
		clientSecrets.length + (client.credentials?.federatedIdentities?.length ?? 0)
	);

	const oidcService = new OidcService();
	const scimService = new ScimService();
	const backNavigation = backNavigate('/settings/admin/oidc-clients');

	const actions = $derived([
		refreshClientAction(client, reloadClient),
		deleteClientAction(backNavigation.leave)
	]);

	async function reloadClient() {
		// The refreshed metadata replaces the fields it manages, so pending edits to them would be stale
		unsavedChanges.discardAll();
		await invalidateAll();

		client = {
			...data.client,
			allowedUserGroupIds: data.client.allowedUserGroups.map((g) => g.id)
		};
		clientSecrets = data.client.credentials?.secrets ?? [];
		scimServiceProvider = data.scimServiceProvider;
		reloadCount++;
	}

	async function updateClient(updatedClient: OidcClientCreateWithLogo) {
		const dataPromise = oidcService.updateClient(client.id, updatedClient);
		const imagePromise =
			updatedClient.logo !== undefined
				? oidcService.updateClientLogo(client, updatedClient.logo, true)
				: Promise.resolve();

		const darkImagePromise =
			updatedClient.darkLogo !== undefined
				? oidcService.updateClientLogo(client, updatedClient.darkLogo, false)
				: Promise.resolve();

		client.isPublic = updatedClient.isPublic;

		const [savedClient] = await Promise.all([dataPromise, imagePromise, darkImagePromise]);
		Object.assign(client, savedClient);

		if (updatedClient.logoUrl || updatedClient.darkLogoUrl) {
			cachedOidcClientLogo.bustCache(client.id);
		}

		// Update the hasLogo and hasDarkLogo flags after successful upload
		if (updatedClient.logo !== undefined || updatedClient.logoUrl !== undefined) {
			client.hasLogo = updatedClient.logo !== null || !!updatedClient.logoUrl;
		}
		if (updatedClient.darkLogo !== undefined || updatedClient.darkLogoUrl !== undefined) {
			client.hasDarkLogo = updatedClient.darkLogo !== null || !!updatedClient.darkLogoUrl;
		}
		if (updatedClient.pkceEnabled) {
			client.pkceEnabled = updatedClient.pkceEnabled;
		}
	}

	async function updateTokenLifetimes(lifetimes: OidcClientTokenLifetimes) {
		await updateClient({ ...client, ...lifetimes });
		client.accessTokenDurationMinutes = lifetimes.accessTokenDurationMinutes;
		client.refreshTokenDurationMinutes = lifetimes.refreshTokenDurationMinutes;
	}

	async function updateFederatedCredentials(federatedIdentities: OidcClientFederatedIdentity[]) {
		// Secrets are read-only in this request, but they are carried over so the client object keeps matching what the server has
		const credentials: OidcClientCredentials = { federatedIdentities, secrets: clientSecrets };
		await updateClient({ ...client, credentials });
		client.credentials = credentials;
	}

	async function saveScimServiceProvider(provider: ScimServiceProviderCreate | null) {
		if (!provider) {
			await scimService.deleteServiceProvider(scimServiceProvider!.id);
			scimServiceProvider = undefined;
			return;
		}
		scimServiceProvider = scimServiceProvider
			? await scimService.updateServiceProvider(scimServiceProvider.id, provider)
			: await scimService.createServiceProvider(provider);
	}

	onDestroy(() => clientSecretStore.clear());
</script>

<svelte:head>
	<title>{m.oidc_client_name({ name: client.name })}</title>
</svelte:head>

{#if client.pkceSupported && !client.pkceEnabled}
	<Alert.Root variant="info">
		<LucideInfo class="size-4" />
		<Alert.Title>{m.pkce_supported_client_title()}</Alert.Title>
		<Alert.Description>
			{m.pkce_supported_client_description()}
		</Alert.Description>
	</Alert.Root>
{/if}

{#if client.clientType === 'cimd'}
	<Alert.Root variant="info">
		<LucideInfo class="size-4" />
		<Alert.Title>{m.cimd_client_managed_fields_title()}</Alert.Title>
		<Alert.Description>
			{m.cimd_client_managed_fields_description()}
		</Alert.Description>
	</Alert.Root>
{/if}

<div class="flex items-center justify-between gap-4">
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
	<div class="flex items-center gap-2">
		<Button variant="outline" size="sm" onclick={() => (showPreview = true)}>
			<LucideEye class="mr-2 size-4" />
			{m.oidc_data_preview()}
		</Button>
		<ActionsMenu item={client} {actions} label={m.actions()} variant="outline" size="icon-sm" />
	</div>
</div>

{#key reloadCount}
	<Tabs.Root value="general" useHash class="gap-6">
		<div class="[scrollbar-width:none] overflow-x-auto border-b">
			<Tabs.List variant="line" class="min-w-max">
				<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
				<Tabs.Trigger value="access">
					{m.access()}
					{#if client.isGroupRestricted && client.allowedUserGroupIds.length === 0}
						<span class="size-1.5 rounded-full bg-yellow-500"></span>
					{/if}
				</Tabs.Trigger>
				<Tabs.Trigger value="credentials">
					{m.credentials()}
					<Badge variant="secondary" class="text-muted-foreground h-4.5 px-1.5 text-[11px]">
						{credentialCount}
					</Badge>
				</Tabs.Trigger>
				<Tabs.Trigger value="scim">{m.scim_provisioning()}</Tabs.Trigger>
			</Tabs.List>
		</div>

		<Tabs.Content value="general" class="flex flex-col gap-6">
			<OidcClientConnectionDetailsCard
				{client}
				secrets={clientSecrets}
				oidcConfiguration={data.oidcConfiguration}
			/>
			<OidcForm existingClient={client} callback={updateClient} />
			<OidcClientTokenLifetimesCard {client} callback={updateTokenLifetimes} />
		</Tabs.Content>

		<Tabs.Content value="access" class="flex flex-col gap-6">
			<OidcClientAllowedUserGroupsCard bind:client />
			<ApiAccessCard clientId={client.id} isPublicClient={client.isPublic} />
		</Tabs.Content>

		<Tabs.Content value="credentials" class="flex flex-col gap-6">
			<OidcClientSecretsCard {client} bind:secrets={clientSecrets} />
			<OidcClientFederatedCredentialsCard {client} callback={updateFederatedCredentials} />
		</Tabs.Content>

		<Tabs.Content value="scim">
			<Card.Root>
				<Card.Header>
					<Card.Title>{m.scim_provisioning()}</Card.Title>
					<Card.Description>
						<FormattedMessage message={m.scim_provisioning_description} />
					</Card.Description>
				</Card.Header>
				<Card.Content>
					<ScimResourceProviderForm
						oidcClientId={client.id}
						existingProvider={scimServiceProvider}
						onSave={saveScimServiceProvider}
					/>
				</Card.Content>
			</Card.Root>
		</Tabs.Content>
	</Tabs.Root>
{/key}

<OidcClientPreviewModal bind:open={showPreview} clientId={client.id} />

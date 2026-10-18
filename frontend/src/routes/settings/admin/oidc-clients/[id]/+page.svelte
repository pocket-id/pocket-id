<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import ActionsMenu from '#lib/components/actions-menu.svelte';
	import AdvancedDisclosure from '#lib/components/advanced-disclosure.svelte';
	import CopyButton from '#lib/components/copy-button.svelte';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcService from '#lib/services/oidc-service.ts';
	import ScimService from '#lib/services/scim-service.ts';
	import clientSecretStore, { autoCreatedSecretId } from '#lib/stores/client-secret-store.ts';
	import unsavedChanges from '#lib/stores/unsaved-changes-store.svelte.ts';
	import type {
		OidcClientCreateWithLogo,
		OidcClientCredentials,
		OidcClientFederatedIdentity,
		OidcClientSecret
	} from '#lib/types/oidc.type.ts';
	import type { ScimServiceProviderCreate } from '#lib/types/scim.type.ts';
	import { cachedOidcClientLogo } from '#lib/utils/cached-image-util.ts';
	import { LucideEye, LucideInfo } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { backNavigate } from '../../users/navigate-back-util';
	import { deleteClientAction, refreshClientAction, typeLabel } from '../oidc-client-actions';
	import OidcForm from '../oidc-client-form.svelte';
	import OidcClientPreviewSheet from '../oidc-client-preview-sheet.svelte';
	import ApiAccessSection from './api-access-section.svelte';
	import FederatedCredentialsSection from './federated-credentials-section.svelte';
	import ScimSection from './scim-section.svelte';
	import SecretReveal from './secret-reveal.svelte';
	import SecretsSection from './secrets-section.svelte';
	import WhoCanSignInSection from './who-can-sign-in-section.svelte';

	let { data } = $props();
	let client = $state({
		...data.client,
		allowedUserGroupIds: data.client.allowedUserGroups.map((g) => g.id)
	});
	// Secrets are managed by their own endpoints, so they are kept out of the client object that the forms below submit
	let clientSecrets = $state<OidcClientSecret[]>(data.client.credentials?.secrets ?? []);
	let apiGrants = $state(data.apiGrants);

	let scimServiceProvider = $state(data.scimServiceProvider);
	let showPreview = $state(false);
	let tab = $state('general');
	// Bumped after the client was reloaded so the forms, which only read the client on mount, pick up the new values
	let reloadCount = $state(0);

	const oidcService = new OidcService();
	const scimService = new ScimService();
	const backNavigation = backNavigate('/settings/admin/oidc-clients');

	const federatedCount = $derived(client.credentials?.federatedIdentities?.length ?? 0);
	const credentialCount = $derived(clientSecrets.length + federatedCount);
	const noOneCanSignIn = $derived(
		client.isGroupRestricted && client.allowedUserGroupIds.length === 0
	);

	// The secret that was generated together with the client is only known until the page is left
	const createdSecret = $derived(
		$autoCreatedSecretId && clientSecrets.some((secret) => secret.id === $autoCreatedSecretId)
			? $clientSecretStore[$autoCreatedSecretId]
			: undefined
	);

	const issuer = $derived(data.oidcConfiguration.issuer);
	const facts = $derived([
		{
			label: m.details(),
			items: [
				{ label: m.client_id(), value: clientIdFact },
				{ label: m.discovery_url(), value: discoveryFact }
			]
		},
		{
			label: m.endpoints(),
			items: [
				{ label: m.issuer(), value: issuerFact },
				{ label: m.authorization(), value: authorizationFact },
				{ label: m.token(), value: tokenFact },
				{ label: m.userinfo(), value: userinfoFact },
				{ label: m.end_session(), value: endSessionFact },
				{ label: m.jwks(), value: jwksFact }
			]
		}
	]);

	const menuActions = $derived([
		refreshClientAction(client, reloadClient),
		deleteClientAction(backNavigation.leave, m.delete_client_ellipsis())
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
		apiGrants = data.apiGrants;
		reloadCount++;
	}

	async function updateClient(updatedClient: OidcClientCreateWithLogo) {
		const [savedClient] = await Promise.all([
			oidcService.updateClient(client.id, updatedClient),
			updatedClient.logo !== undefined
				? oidcService.updateClientLogo(client, updatedClient.logo, true)
				: undefined,
			updatedClient.darkLogo !== undefined
				? oidcService.updateClientLogo(client, updatedClient.darkLogo, false)
				: undefined
		]);
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

<!-- Endpoints under the issuer show only their path so each one reads whole in its column, while the tooltip and the copy button carry the full URL -->
{#snippet fact(value: string, copyLabel: string, testId?: string)}
	<span class="flex min-w-0 items-center gap-1">
		<span class="text-mono-s truncate font-mono" title={value} data-testid={testId}>
			{value !== issuer && value.startsWith(issuer) ? value.slice(issuer.length) : value}
		</span>
		<CopyButton {value} label={copyLabel} />
	</span>
{/snippet}
{#snippet clientIdFact()}{@render fact(client.id, m.copy_client_id(), 'client-id')}{/snippet}
{#snippet discoveryFact()}
	{@render fact(`${issuer}/.well-known/openid-configuration`, m.copy_discovery_url())}
{/snippet}
{#snippet issuerFact()}{@render fact(issuer, m.copy_issuer_url())}{/snippet}
{#snippet authorizationFact()}
	{@render fact(data.oidcConfiguration.authorization_endpoint, m.copy_authorization_url())}
{/snippet}
{#snippet tokenFact()}{@render fact(
		data.oidcConfiguration.token_endpoint,
		m.copy_token_url()
	)}{/snippet}
{#snippet userinfoFact()}
	{@render fact(data.oidcConfiguration.userinfo_endpoint, m.copy_userinfo_url())}
{/snippet}
{#snippet endSessionFact()}
	{@render fact(data.oidcConfiguration.end_session_endpoint, m.copy_end_session_url())}
{/snippet}
{#snippet jwksFact()}{@render fact(data.oidcConfiguration.jwks_uri, m.copy_jwks_url())}{/snippet}

<PageHeader
	title={client.name}
	breadcrumbs={[
		{ label: m.administration() },
		{ label: m.oidc_clients(), href: '/settings/admin/oidc-clients' }
	]}
	{facts}
>
	{#snippet identity()}
		<OidcClientAvatar
			id={client.id}
			name={client.name}
			hasLogo={client.hasLogo || client.hasDarkLogo}
			size={56}
		/>
	{/snippet}
	{#snippet status()}
		{#if noOneCanSignIn}
			<Badge variant="warning">{m.no_one_can_sign_in()}</Badge>
		{:else if client.isGroupRestricted}
			<Badge variant="muted">
				{client.allowedUserGroupIds.length === 1
					? m.limited_to_one_group()
					: m.limited_to_count_groups({ count: client.allowedUserGroupIds.length })}
			</Badge>
		{/if}
	{/snippet}
	{#snippet tags()}
		<Badge variant="outline">{typeLabel(client)}</Badge>
		{#if client.clientType === 'cimd'}
			<Badge variant="outline">{m.metadata_document()}</Badge>
		{/if}
	{/snippet}
	{#snippet actions()}
		<Button variant="secondary" onclick={() => (showPreview = true)}>
			<LucideEye />
			<span class="max-sm:sr-only">{m.data_preview()}</span>
		</Button>
		<ActionsMenu item={client} actions={menuActions} label={m.more_actions()} variant="outline" />
	{/snippet}
</PageHeader>

{#key reloadCount}
	<Tabs.Root bind:value={tab} useHash class="mt-4 gap-8">
		<div class="flex shadow-[inset_0_-1px_0_var(--border)]">
			<div class="scroll-fade-x min-w-0 [scrollbar-width:none] overflow-x-auto">
				<Tabs.List
					variant="line"
					class="shadow-none max-sm:gap-4"
					aria-label={m.name_sections({ name: client.name })}
				>
					<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
					<Tabs.Trigger value="access">
						{m.access()}
						{#if noOneCanSignIn}
							<span class="bg-warning-solid size-1.5 rounded-full" aria-hidden="true"></span>
							<span class="sr-only">{m.no_one_can_sign_in()}</span>
						{/if}
					</Tabs.Trigger>
					<Tabs.Trigger value="credentials">
						{m.credentials()}
						<span class="text-fg-muted tabular-nums">{credentialCount}</span>
					</Tabs.Trigger>
					{#if data.features?.scim.enabled}
						<Tabs.Trigger value="scim">{m.scim()}</Tabs.Trigger>
					{/if}
				</Tabs.List>
			</div>
		</div>

		<Tabs.Content value="general" class="flex flex-col gap-10">
			{#if client.clientType === 'cimd'}
				<Alert.Root variant="info">
					<LucideInfo />
					<Alert.Title>{m.some_settings_come_from_a_metadata_document()}</Alert.Title>
					<Alert.Description>
						{m.cimd_client_read_only_description({ name: client.name })}
					</Alert.Description>
				</Alert.Root>
			{/if}
			{#if createdSecret}
				<SecretReveal
					label={m.client_secret()}
					value={createdSecret}
					testId="created-client-secret"
				/>
			{/if}
			<OidcForm existingClient={client} callback={updateClient} />
		</Tabs.Content>

		<Tabs.Content value="access" class="flex flex-col gap-10">
			<WhoCanSignInSection bind:client />
			{#if data.features?.apis.enabled}
				<AdvancedDisclosure
					summary={m.apis_name_may_request_tokens_for({ name: client.name })}
					changedCount={apiGrants.length ? 1 : 0}
				>
					<ApiAccessSection {client} bind:grants={apiGrants} />
				</AdvancedDisclosure>
			{/if}
		</Tabs.Content>

		<Tabs.Content value="credentials" class="flex flex-col gap-10">
			<SecretsSection {client} bind:secrets={clientSecrets} />
			{#if data.features?.federatedCredentials.enabled}
				<AdvancedDisclosure
					summary={m.federated_credentials_summary()}
					changedCount={federatedCount ? 1 : 0}
				>
					<FederatedCredentialsSection {client} callback={updateFederatedCredentials} />
				</AdvancedDisclosure>
			{/if}
		</Tabs.Content>

		<Tabs.Content value="scim">
			<ScimSection
				{client}
				existingProvider={scimServiceProvider}
				onSave={saveScimServiceProvider}
			/>
		</Tabs.Content>
	</Tabs.Root>
{/key}

<OidcClientPreviewSheet bind:open={showPreview} {client} />

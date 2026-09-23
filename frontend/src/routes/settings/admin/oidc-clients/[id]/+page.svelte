<script lang="ts">
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import FormattedMessage from '$lib/components/formatted-message.svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Field from '$lib/components/ui/field';
	import * as Tabs from '$lib/components/ui/tabs';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import ScimService from '$lib/services/scim-service';
	import clientSecretStore, { autoCreatedSecretId } from '$lib/stores/client-secret-store';
	import type {
		OidcClientCreateWithLogo,
		OidcClientCredentials,
		OidcClientFederatedIdentity,
		OidcClientSecret,
		OidcClientTokenLifetimes
	} from '$lib/types/oidc.type';
	import type { ScimServiceProviderCreate } from '$lib/types/scim.type';
	import { cachedOidcClientLogo } from '$lib/utils/cached-image-util';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { trackUnsavedValue } from '$lib/utils/unsaved-changes-util.svelte';
	import { LucideChevronLeft, LucideInfo, LucideTriangleAlert } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import { backNavigate } from '../../users/navigate-back-util';
	import OidcForm from '../oidc-client-form.svelte';
	import OidcClientPreviewModal from '../oidc-client-preview-modal.svelte';
	import ApiAccessCard from './api-access-card.svelte';
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
	let showAllDetails = $state(false);
	let showPreview = $state(false);

	const oidcService = new OidcService();
	const scimService = new ScimService();
	const backNavigation = backNavigate('/settings/admin/oidc-clients');

	const allowedUserGroups = trackUnsavedValue(
		() => client.allowedUserGroupIds,
		(allowedUserGroupIds) => {
			client.allowedUserGroupIds = allowedUserGroupIds;
		},
		(allowedUserGroupIds) => oidcService.updateAllowedUserGroups(client.id, allowedUserGroupIds)
	);

	const setupDetails = $state({
		[m.issuer_url()]: data.oidcConfiguration.issuer,
		[m.authorization_url()]: data.oidcConfiguration.authorization_endpoint,
		[m.oidc_discovery_url()]: `${data.oidcConfiguration.issuer}/.well-known/openid-configuration`,
		[m.token_url()]: data.oidcConfiguration.token_endpoint,
		[m.userinfo_url()]: data.oidcConfiguration.userinfo_endpoint,
		[m.logout_url()]: data.oidcConfiguration.end_session_endpoint,
		[m.certificate_url()]: data.oidcConfiguration.jwks_uri,
		[m.pkce()]: client.pkceEnabled ? m.enabled() : m.disabled()
	});

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
		setupDetails[m.pkce()] = updatedClient.pkceEnabled ? m.enabled() : m.disabled();
		setupDetails[m.requires_reauthentication()] = updatedClient.requiresReauthentication
			? m.enabled()
			: m.disabled();

		const [savedClient] = await Promise.all([dataPromise, imagePromise, darkImagePromise]);
		Object.assign(client, savedClient);

		setupDetails[m.requires_pushed_authorization_requests()] =
			updatedClient.requiresPushedAuthorizationRequests ? m.enabled() : m.disabled();
		if (updatedClient.logoUrl) {
			cachedOidcClientLogo.bustCache(client.id, true);
		}
		if (updatedClient.darkLogoUrl) {
			cachedOidcClientLogo.bustCache(client.id, false);
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

	async function enableGroupRestriction() {
		client.isGroupRestricted = true;
		await oidcService
			.updateClient(client.id, {
				...client,
				isGroupRestricted: true
			})
			.then(() => {
				toast.success(m.user_groups_restriction_updated_successfully());
				client.isGroupRestricted = true;
			})
			.catch(axiosErrorToast);
	}

	function disableGroupRestriction() {
		openConfirmDialog({
			title: m.unrestrict_oidc_client({ clientName: client.name }),
			message: {
				message: m.confirm_unrestrict_oidc_client_description,
				inputs: { clientName: client.name }
			},
			confirm: {
				label: m.unrestrict(),
				destructive: true,
				action: async () => {
					await oidcService
						.updateClient(client.id, {
							...client,
							isGroupRestricted: false
						})
						.then(() => {
							toast.success(m.user_groups_restriction_updated_successfully());
							client.allowedUserGroupIds = [];
							allowedUserGroups.markSaved();
							client.isGroupRestricted = false;
						})
						.catch(axiosErrorToast);
				}
			}
		});
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

{#snippet UnrestrictButton()}
	<Button
		onclick={enableGroupRestriction}
		variant={client.isGroupRestricted ? 'secondary' : 'default'}>{m.restrict()}</Button
	>
{/snippet}

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

<div>
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
</div>

<Tabs.Root value="general" useHash class="gap-4">
	<div class="overflow-x-auto pb-1">
		<Tabs.List variant="line" class="min-w-max">
			<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
			<Tabs.Trigger value="user-groups">
				{m.allowed_user_groups()}
				{#if client.isGroupRestricted && client.allowedUserGroupIds.length === 0}
					<LucideTriangleAlert class="ml-0.5 size-4 text-yellow-600 dark:text-yellow-400" />
				{/if}</Tabs.Trigger
			>
			<Tabs.Trigger value="credentials">{m.credentials()}</Tabs.Trigger>
			<Tabs.Trigger value="api-access">{m.api_access()}</Tabs.Trigger>
			<Tabs.Trigger value="scim">{m.scim_provisioning()}</Tabs.Trigger>
			<Tabs.Trigger value="preview">{m.oidc_data_preview()}</Tabs.Trigger>
		</Tabs.List>
	</div>

	<Tabs.Content value="general" class="flex flex-col gap-4">
		<Card.Root>
			<Card.Header>
				<Card.Title>{client.name}</Card.Title>
			</Card.Header>
			<Card.Content>
				<div class="flex flex-col">
					<div class="mb-2 flex flex-col sm:flex-row sm:items-center">
						<Field.Label class="w-52">{m.client_id()}</Field.Label>
						<CopyToClipboard value={client.id}>
							<span class="text-muted-foreground text-sm" data-testid="client-id">
								{client.id}
							</span>
						</CopyToClipboard>
					</div>
					{#if $autoCreatedSecretId && clientSecrets.some((secret) => secret.id === $autoCreatedSecretId) && $clientSecretStore[$autoCreatedSecretId]}
						<div class="mb-2 flex flex-col sm:flex-row sm:items-center">
							<Field.Label class="w-52">{m.client_secret()}</Field.Label>
							<CopyToClipboard value={$clientSecretStore[$autoCreatedSecretId]}>
								<span class="text-muted-foreground text-sm break-all" data-testid="client-secret">
									{$clientSecretStore[$autoCreatedSecretId]}
								</span>
							</CopyToClipboard>
						</div>
					{/if}
					{#if showAllDetails}
						<div transition:slide>
							{#each Object.entries(setupDetails) as [key, value] (key)}
								<div class="mb-2 flex flex-col sm:flex-row sm:items-center">
									<Field.Label class="w-52">{key}</Field.Label>
									<CopyToClipboard {value}>
										<span class="text-muted-foreground text-sm">{value}</span>
									</CopyToClipboard>
								</div>
							{/each}
						</div>
					{/if}

					{#if !showAllDetails}
						<div class="mt-4 flex justify-center">
							<Button onclick={() => (showAllDetails = true)} size="sm" variant="ghost"
								>{m.show_more_details()}</Button
							>
						</div>
					{/if}
				</div>
			</Card.Content>
		</Card.Root>

		<Card.Root>
			<Card.Content>
				<OidcForm mode="update" existingClient={client} callback={updateClient} />
			</Card.Content>
		</Card.Root>

		<OidcClientTokenLifetimesCard {client} callback={updateTokenLifetimes} />
	</Tabs.Content>

	<Tabs.Content value="credentials" id="credentials" class="flex flex-col gap-4">
		<OidcClientSecretsCard {client} bind:secrets={clientSecrets} />

		<OidcClientFederatedCredentialsCard {client} callback={updateFederatedCredentials} />
	</Tabs.Content>

	<Tabs.Content value="user-groups" id="allowed-user-groups">
		<Card.Root>
			<Card.Header>
				<div class="flex items-center justify-between gap-4">
					<div>
						<Card.Title>{m.allowed_user_groups()}</Card.Title>
						<Card.Description>
							{client.isGroupRestricted
								? m.allowed_user_groups_description()
								: m.allowed_user_groups_status_unrestricted_description()}
						</Card.Description>
					</div>
					{#if !client.isGroupRestricted}
						{@render UnrestrictButton()}
					{/if}
				</div>
			</Card.Header>
			{#if client.isGroupRestricted}
				<Card.Content>
					<UserGroupSelection bind:selectedGroupIds={client.allowedUserGroupIds} />
					<div class="mt-5 flex justify-end gap-3">
						<Button onclick={disableGroupRestriction} variant="secondary">{m.unrestrict()}</Button>
					</div>
				</Card.Content>
			{/if}
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="api-access" id="api-access">
		<ApiAccessCard clientId={client.id} isPublicClient={client.isPublic} />
	</Tabs.Content>

	<Tabs.Content value="scim" id="scim-provisioning">
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

	<Tabs.Content value="preview">
		<Card.Root>
			<Card.Header>
				<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
					<div>
						<Card.Title>
							{m.oidc_data_preview()}
						</Card.Title>
						<Card.Description>
							{m.preview_the_oidc_data_that_would_be_sent_for_different_users()}
						</Card.Description>
					</div>

					<Button variant="outline" onclick={() => (showPreview = true)}>
						{m.show()}
					</Button>
				</div>
			</Card.Header>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>
<OidcClientPreviewModal bind:open={showPreview} clientId={client.id} />

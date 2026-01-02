<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import CollapsibleCard from '$lib/components/collapsible-card.svelte';
	import { openConfirmDialog } from '$lib/components/confirm-dialog';
	import CopyToClipboard from '$lib/components/copy-to-clipboard.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import * as Field from '$lib/components/ui/field';
	import UserGroupSelection from '$lib/components/user-group-selection.svelte';
	import { m } from '$lib/paraglide/messages';
	import OidcService from '$lib/services/oidc-service';
	import ScimService from '$lib/services/scim-service';
	import clientSecretStore from '$lib/stores/client-secret-store';
	import type { OidcClientCreateWithLogo } from '$lib/types/oidc.type';
	import type { ScimServiceProviderCreate } from '$lib/types/scim.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucideChevronLeft, LucideRefreshCcw } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { slide } from 'svelte/transition';
	import { backNavigate } from '../../users/navigate-back-util';
	import OidcForm from '../oidc-client-form.svelte';
	import OidcClientPreviewModal from '../oidc-client-preview-modal.svelte';
	import ScimResourceProviderForm from './scim-resource-provider-form.svelte';

	let { data } = $props();
	let client = $state({
		...data.client,
		allowedUserGroupIds: data.client.allowedUserGroups.map((g) => g.id)
	});

	let scimServiceProvider = $state(data.scimServiceProvider);
	let showAllDetails = $state(false);
	let showPreview = $state(false);

	const oidcService = new OidcService();
	const scimService = new ScimService();
	const backNavigation = backNavigate('/settings/admin/oidc-clients');

	const setupDetails = $state({
		[m.authorization_url()]: `https://${page.url.host}/authorize`,
		[m.oidc_discovery_url()]: `https://${page.url.host}/.well-known/openid-configuration`,
		[m.token_url()]: `https://${page.url.host}/api/oidc/token`,
		[m.userinfo_url()]: `https://${page.url.host}/api/oidc/userinfo`,
		[m.logout_url()]: `https://${page.url.host}/api/oidc/end-session`,
		[m.certificate_url()]: `https://${page.url.host}/.well-known/jwks.json`,
		[m.pkce()]: client.pkceEnabled ? m.enabled() : m.disabled(),
		[m.requires_reauthentication()]: client.requiresReauthentication ? m.enabled() : m.disabled()
	});

	async function updateClient(updatedClient: OidcClientCreateWithLogo) {
		let success = true;
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

		await Promise.all([dataPromise, imagePromise, darkImagePromise])
			.then(() => {
				// Update the hasLogo and hasDarkLogo flags after successful upload
				if (updatedClient.logo !== undefined) {
					client.hasLogo = updatedClient.logo !== null || !!updatedClient.logoUrl;
				}
				if (updatedClient.darkLogo !== undefined) {
					client.hasDarkLogo = updatedClient.darkLogo !== null || !!updatedClient.darkLogoUrl;
				}
				toast.success(m.oidc_client_updated_successfully());
			})
			.catch((e) => {
				axiosErrorToast(e);
				success = false;
			});

		return success;
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
			message: m.confirm_unrestrict_oidc_client_description({ clientName: client.name }),
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
							client.isGroupRestricted = false;
						})
						.catch(axiosErrorToast);
				}
			}
		});
	}

	async function createClientSecret() {
		openConfirmDialog({
			title: m.create_new_client_secret(),
			message: m.are_you_sure_you_want_to_create_a_new_client_secret(),
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

	async function saveScimServiceProvider(provider: ScimServiceProviderCreate | null) {
		try {
			if (!provider) {
				await scimService.deleteServiceProvider(scimServiceProvider!.id);
				scimServiceProvider = undefined;
				toast.success(m.scim_disabled_successfully());
				return true;
			}
			let createdProvider;
			if (scimServiceProvider) {
				createdProvider = await scimService.updateServiceProvider(scimServiceProvider.id, provider);
				toast.success(m.scim_configuration_updated_successfully());
			} else {
				createdProvider = await scimService.createServiceProvider(provider);
				toast.success(m.scim_enabled_successfully());
			}
			scimServiceProvider = createdProvider;
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	beforeNavigate(() => {
		clientSecretStore.clear();
	});
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

<div>
	<button type="button" class="text-muted-foreground flex text-sm" onclick={backNavigation.go}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>{client.name}</Card.Title>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-col">
			<div class="mb-2 flex flex-col sm:flex-row sm:items-center">
				<Field.Label class="w-50">{m.client_id()}</Field.Label>
				<CopyToClipboard value={client.id}>
					<span class="text-muted-foreground text-sm" data-testid="client-id"> {client.id}</span>
				</CopyToClipboard>
			</div>
			{#if !client.isPublic}
				<div class="mt-1 mb-2 flex flex-col sm:flex-row sm:items-center">
					<Field.Label class="w-50">{m.client_secret()}</Field.Label>
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
								aria-label="Create new client secret"><LucideRefreshCcw class="size-3" /></Button
							>
						</div>
					{/if}
				</div>
			{/if}
			{#if showAllDetails}
				<div transition:slide>
					{#each Object.entries(setupDetails) as [key, value]}
						<div class="mb-5 flex flex-col sm:flex-row sm:items-center">
							<Field.Label class="w-50">{key}</Field.Label>
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
<CollapsibleCard
	id="allowed-user-groups"
	title={m.allowed_user_groups()}
	button={!client.isGroupRestricted ? UnrestrictButton : undefined}
	forcedExpanded={client.isGroupRestricted ? undefined : false}
	description={client.isGroupRestricted
		? m.allowed_user_groups_description()
		: m.allowed_user_groups_status_unrestricted_description()}
>
	<UserGroupSelection
		bind:selectedGroupIds={client.allowedUserGroupIds}
		selectionDisabled={!client.isGroupRestricted}
	/>
	<div class="mt-5 flex justify-end gap-3">
		<Button onclick={disableGroupRestriction} variant="secondary">{m.unrestrict()}</Button>

		<Button usePromiseLoading onclick={() => updateUserGroupClients(client.allowedUserGroupIds)}
			>{m.save()}</Button
		>
	</div>
</CollapsibleCard>
<CollapsibleCard
	id="scim-provisioning"
	title={m.scim_provisioning()}
	description={m.scim_provisioning_description()}
>
	<ScimResourceProviderForm
		oidcClientId={client.id}
		existingProvider={scimServiceProvider}
		onSave={saveScimServiceProvider}
	/>
</CollapsibleCard>
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
<OidcClientPreviewModal bind:open={showPreview} clientId={client.id} />

<script lang="ts">
	import * as Card from '#lib/components/ui/card/index.ts';
	import * as Tabs from '#lib/components/ui/tabs/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideChevronLeft } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { backNavigate } from '../../users/navigate-back-util';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type { OidcClaimMappingPolicy } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import OidcClaimMappingPolicyForm from '../oidc-claim-mapping-policy-form.svelte';
	import OidcClaimMappingsCard from './oidc-claim-mappings-card.svelte';
	import OidcClaimMappingPolicyClientsTab from './oidc-claim-mapping-policy-clients-tab.svelte';
	import Button from '#lib/components/ui/button/button.svelte';
	import OidcClientPreviewModal from '../../oidc-clients/oidc-client-preview-modal.svelte';

	let { data } = $props();
	let policy: OidcClaimMappingPolicy = $state(data.policy);
	let clientsTab = $state<OidcClaimMappingPolicyClientsTab>();
	let showPreview = $state(false);

	const claimMappingPolicyService = new OidcClaimMappingPolicyService();
	const backNavigation = backNavigate('/settings/admin/oidc-claim-mapping-policies');

	async function updatePolicy(updatedPolicy: OidcClaimMappingPolicy) {
		if (!policy.id) return;

		try {
			policy = await claimMappingPolicyService.updateClaimMappingPolicy(policy.id, updatedPolicy);
			toast.success(m.claim_mapping_policy_updated_successfully());
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	let clientId: string = $state('');
	async function fetchClientForPreview() {
		await claimMappingPolicyService
			.listClients(policy.id!, {
				pagination: {
					page: 0,
					limit: 1
				}
			})
			.then((data) => (clientId = data.data[0]?.id || ''));
		if (clientId == '' && policy.isDefault) {
			await claimMappingPolicyService
				.listAssignableClients(policy.id!, {
					pagination: {
						page: 0,
						limit: 1
					}
				})
				.then((data) => (clientId = data.data[0]?.id || ''));
		}
	}
</script>

<svelte:head>
	<title>{m.edit_claim_mapping_policy()}</title>
</svelte:head>

<div class="flex items-center justify-between">
	<button class="text-muted-foreground flex text-sm" onclick={() => backNavigation.go()}
		><LucideChevronLeft class="size-5" /> {m.back()}</button
	>
</div>

<Tabs.Root value="general" useHash class="gap-4">
	<div class="overflow-x-auto pb-1">
		<Tabs.List variant="line" class="min-w-max">
			<Tabs.Trigger value="general">{m.general()}</Tabs.Trigger>
			<Tabs.Trigger value="clients">{m.oidc_clients()}</Tabs.Trigger>
			<Tabs.Trigger value="preview" onclick={fetchClientForPreview}
				>{m.oidc_data_preview()}</Tabs.Trigger
			>
		</Tabs.List>
	</div>

	<Tabs.Content value="general" class="flex flex-col gap-4">
		<Card.Root>
			<Card.Header>
				<Card.Title>{m.general()}</Card.Title>
			</Card.Header>

			<Card.Content>
				<OidcClaimMappingPolicyForm existingPolicy={policy} callback={updatePolicy} mode="update" />
			</Card.Content>
		</Card.Root>

		<OidcClaimMappingsCard {policy} onClaimMappingsSave={updatePolicy} />
	</Tabs.Content>

	<Tabs.Content value="clients" class="flex flex-col gap-4">
		<Card.Root>
			<Card.Header>
				<div class="flex flex-wrap items-center justify-between gap-4 md:flex-nowrap">
					<div>
						<Card.Title>{m.oidc_clients()}</Card.Title>
						<Card.Description>{m.claim_mapping_policy_clients_description()}</Card.Description>
					</div>
					<Button class="w-full md:w-auto" onclick={() => clientsTab?.openPicker()}>
						{m.assign_client()}
					</Button>
				</div>
			</Card.Header>
			<Card.Content>
				<OidcClaimMappingPolicyClientsTab bind:this={clientsTab} policyId={policy.id!} />
			</Card.Content>
		</Card.Root>
	</Tabs.Content>

	<Tabs.Content value="preview" class="flex flex-col gap-4">
		<Card.Root>
			<Card.Header>
				<div class="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
					<div>
						<Card.Title>
							{m.oidc_data_preview()}
						</Card.Title>
						<Card.Description>
							{m.preview_the_oidc_data_that_would_be_sent_for_different_users()}
							{#if clientId === ''}
								<p class="text-muted-foreground mt-3 text-sm">
									{m.claim_mapping_policy_preview_disabled()}
								</p>
							{/if}
						</Card.Description>
					</div>

					<Button variant="outline" onclick={() => (showPreview = true)} disabled={clientId === ''}>
						{m.show()}
					</Button>
				</div>
			</Card.Header>
		</Card.Root>
	</Tabs.Content>
</Tabs.Root>

<OidcClientPreviewModal bind:open={showPreview} {clientId} />

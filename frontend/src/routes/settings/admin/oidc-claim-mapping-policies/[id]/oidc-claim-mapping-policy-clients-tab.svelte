<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Button } from '#lib/components/ui/button/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';
	import { encodeClientIdParam } from '#lib/utils/client-id-util.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import ClientSelectionModal from './client-selection-modal.svelte';

	let { policyId }: { policyId: string } = $props();

	const oidcClaimMappingPolicyService = new OidcClaimMappingPolicyService();

	let tableRef: AdvancedTable<OidcClientMetaData>;
	let pickerOpen = $state(false);

	const columns: AdvancedTableColumn<OidcClientMetaData>[] = [
		{ label: m.client(), key: 'client', cell: ClientCell },
		{
			label: m.client_type(),
			column: 'clientType',
			sortable: true,
			value: (item) =>
				item.clientType === 'cimd' ? m.client_type_metadata_document() : m.client_type_standard()
		},
		{ label: '', key: 'actions', cell: ActionsCell }
	];

	function fetchCallback(options: ListRequestOptions) {
		return oidcClaimMappingPolicyService.listClients(policyId, options);
	}

	export async function refresh() {
		await tableRef?.refresh();
	}

	export function openPicker() {
		pickerOpen = true;
	}

	async function assignClient(client: OidcClientMetaData) {
		pickerOpen = false;
		try {
			await oidcClaimMappingPolicyService.assignClient(policyId, client.id);
			await tableRef?.refresh();
			toast.success(m.claim_mapping_policy_clients_updated_successfully());
		} catch (e) {
			axiosErrorToast(e);
		}
	}

	function removeClient(client: OidcClientMetaData) {
		openConfirmDialog({
			title: m.unassign_client_from_claim_mapping_policy({ name: client.name }),
			message: m.are_you_sure_you_want_to_detach_this_client_from_the_claim_mapping_policy(),
			confirm: {
				label: m.unassign_client_from_claim_mapping_policy({ name: client.name }),
				destructive: true,
				action: async () => {
					try {
						await oidcClaimMappingPolicyService.removeClient(policyId, client.id);
						await tableRef?.refresh();
						toast.success(m.claim_mapping_policy_clients_updated_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet ClientCell({ item }: { item: OidcClientMetaData })}
	<div class="flex items-center gap-3">
		<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo} />
		<a
			class="font-medium hover:underline"
			href={`/settings/admin/oidc-clients/${encodeClientIdParam(item.id)}`}
		>
			{item.name}
		</a>
	</div>
{/snippet}

{#snippet ActionsCell({ item }: { item: OidcClientMetaData })}
	<div class="flex justify-end gap-1">
		<Button
			variant="ghost"
			size="sm"
			aria-label={m.unassign_client_from_claim_mapping_policy({ name: item.name })}
			onclick={() => removeClient(item)}
		>
			<LucideTrash class="size-4" />
		</Button>
	</div>
{/snippet}

<AdvancedTable
	id={`claim-mapping-policy-clients-${policyId}`}
	bind:this={tableRef}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{fetchCallback}
/>

<ClientSelectionModal bind:open={pickerOpen} {policyId} onSelect={assignClient} />

<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { OidcClaimMappingPolicyMetadataDto } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucideCopy, LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	const oidcClaimMappingPolicyService = new OidcClaimMappingPolicyService();

	let tableRef: AdvancedTable<OidcClaimMappingPolicyMetadataDto>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<OidcClaimMappingPolicyMetadataDto>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.is_default(),
			column: 'isDefault',
			sortable: true,
			value: (item: OidcClaimMappingPolicyMetadataDto) => (item.isDefault ? m.yes() : m.no())
		}
	];

	const actions: CreateAdvancedTableActions<OidcClaimMappingPolicyMetadataDto> = () => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			variant: 'ghost',
			onClick: (policy) => goto(`/settings/admin/oidc-claim-mapping-policies/${policy.id}`)
		},
		{
			label: m.duplicate(),
			icon: LucideCopy,
			onClick: (policy) => {
				duplicateClaimMappingPolicy(policy, `${policy.name} (${m.copy()})`);
			}
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (policy) => deleteClaimMappingPolicy(policy)
		}
	];

	async function duplicateClaimMappingPolicy(
		policy: OidcClaimMappingPolicyMetadataDto,
		name: string
	) {
		try {
			const createdPolicy = await oidcClaimMappingPolicyService.duplicateClaimMappingPolicy(
				policy.id,
				name
			);
			toast.success(m.claim_mapping_policy_created_successfully());
			await goto(`/settings/admin/oidc-claim-mapping-policies/${createdPolicy.id}`);
			return true;
		} catch (e) {
			axiosErrorToast(e);
			return false;
		}
	}

	async function deleteClaimMappingPolicy(policy: OidcClaimMappingPolicyMetadataDto) {
		openConfirmDialog({
			title: m.delete_claim_mapping_policy({ name: policy.name }),
			message: m.are_you_sure_you_want_to_delete_this_claim_mapping_policy(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await oidcClaimMappingPolicyService.deleteClaimMappingPolicy(policy.id);
						await refresh();
						toast.success(m.claim_mapping_policy_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	id="oidc-claim-mapping-policy-list"
	bind:this={tableRef}
	fetchCallback={oidcClaimMappingPolicyService.listClaimMappingPolicies}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{actions}
/>

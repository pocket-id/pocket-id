<script lang="ts">
	import { openConfirmDialog } from '#lib/components/confirm-dialog/index.ts';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '@pocket-id/ui/badge';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { CustomScope } from '#lib/types/oidc-claim-mapping-policy.type.ts';
	import { axiosErrorToast } from '#lib/utils/error-util.ts';
	import { LucidePencil, LucideTrash2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let { onEdit }: { onEdit: (scope: CustomScope) => void } = $props();

	const policyService = new OidcClaimMappingPolicyService();
	let tableRef = $state<AdvancedTable<CustomScope>>();

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<CustomScope>[] = [
		{ label: m.scope(), column: 'key', sortable: true, cell: ScopeCell },
		{ label: m.released_by(), key: 'policies', cell: PoliciesCell }
	];

	const actions: CreateAdvancedTableActions<CustomScope> = () => [
		{ label: m.edit_ellipsis(), primary: true, icon: LucidePencil, onClick: onEdit },
		{ label: m.delete_ellipsis(), icon: LucideTrash2, variant: 'danger', onClick: confirmDelete }
	];

	function confirmDelete(scope: CustomScope) {
		openConfirmDialog({
			title: m.delete_name_question({ name: scope.name }),
			message: m.delete_scope_consequence(),
			confirm: {
				label: m.delete_scope(),
				destructive: true,
				action: async () => {
					try {
						await policyService.deleteCustomScope(scope.id);
						toast.success(m.name_deleted({ name: scope.name }));
						await refresh();
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet ScopeCell({ item }: { item: CustomScope })}
	<div class="flex min-w-0 flex-col">
		<span class="text-control text-fg truncate font-medium">{item.name}</span>
		<span class="text-mono-s text-fg-muted truncate font-mono">{item.key}</span>
	</div>
{/snippet}

{#snippet PoliciesCell({ item }: { item: CustomScope })}
	{#if item.policies?.length}
		<span class="text-control text-fg truncate">{item.policies.join(', ')}</span>
	{:else}
		<Badge variant="warning">{m.not_used_by_any_policy()}</Badge>
	{/if}
{/snippet}

<AdvancedTable
	id="custom-scope-list"
	bind:this={tableRef}
	fetchCallback={policyService.listCustomScopes}
	defaultSort={{ column: 'key', direction: 'asc' }}
	onRowClick={onEdit}
	{columns}
	{actions}
/>

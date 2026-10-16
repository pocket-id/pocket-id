<script lang="ts">
	import OidcClientAvatar from '#lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import OidcClaimMappingPolicyService from '#lib/services/oidc-claim-mapping-policy-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';
	import type { OidcClientMetaData } from '#lib/types/oidc.type.ts';

	let {
		open = $bindable(),
		policyId,
		onSelect
	}: {
		open: boolean;
		policyId: string;
		onSelect: (client: OidcClientMetaData) => void;
	} = $props();

	const oidcClaimMappingPolicyService = new OidcClaimMappingPolicyService();

	const columns: AdvancedTableColumn<OidcClientMetaData>[] = [
		{ label: m.logo(), key: 'logo', cell: LogoCell },
		{ label: m.name(), column: 'name', sortable: true },
		{
			label: m.client_type(),
			column: 'clientType',
			sortable: true,
			value: (item) =>
				item.clientType === 'cimd' ? m.client_type_metadata_document() : m.client_type_standard()
		}
	];

	function fetchCallback(options: ListRequestOptions) {
		return oidcClaimMappingPolicyService.listAssignableClients(policyId, options);
	}
</script>

{#snippet LogoCell({ item }: { item: OidcClientMetaData })}
	<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo} />
{/snippet}

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{m.add_client()}</Dialog.Title>
			<Dialog.Description>
				{m.select_a_client_to_assign_to_this_claim_mapping_policy()}
			</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id="claim-mapping-policy-client-selection"
			onRowClick={(item) => onSelect(item)}
			{fetchCallback}
			defaultSort={{ column: 'name', direction: 'asc' }}
			{columns}
		/>
	</Dialog.Content>
</Dialog.Root>

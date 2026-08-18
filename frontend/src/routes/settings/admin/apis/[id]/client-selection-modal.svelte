<script lang="ts">
	import OidcClientAvatar from '$lib/components/oidc-client-avatar.svelte';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { ApiClient } from '$lib/types/api.type';
	import type { ListRequestOptions } from '$lib/types/list-request.type';

	let {
		open = $bindable(),
		apiId,
		onSelect
	}: {
		open: boolean;
		apiId: string;
		onSelect: (client: ApiClient) => void;
	} = $props();

	const apisService = new ApisService();

	const columns: AdvancedTableColumn<ApiClient>[] = [
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
		return apisService.listAssignableClients(apiId, options);
	}
</script>

{#snippet LogoCell({ item }: { item: ApiClient })}
	<OidcClientAvatar id={item.id} name={item.name} hasLogo={item.hasLogo} />
{/snippet}

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{m.add_client()}</Dialog.Title>
			<Dialog.Description>{m.select_a_client_to_grant_access_to_this_api()}</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id="api-client-selection"
			onRowClick={(item) => onSelect(item)}
			{fetchCallback}
			defaultSort={{ column: 'name', direction: 'asc' }}
			{columns}
		/>
	</Dialog.Content>
</Dialog.Root>

<script lang="ts">
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import * as Dialog from '#lib/components/ui/dialog/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type { AdvancedTableColumn } from '#lib/types/advanced-table.type.ts';
	import type { Api } from '#lib/types/api.type.ts';
	import type { ListRequestOptions } from '#lib/types/list-request.type.ts';

	let {
		open = $bindable(),
		clientId,
		onSelect
	}: {
		open: boolean;
		clientId: string;
		onSelect: (api: Api) => void;
	} = $props();

	const apisService = new ApisService();

	const columns: AdvancedTableColumn<Api>[] = [
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.api_resource(), column: 'resource', sortable: true },
		{ label: m.api_permissions(), key: 'permissions', value: (item) => item.permissions.length }
	];

	function fetchCallback(options: ListRequestOptions) {
		return apisService.listAssignableApis(clientId, options);
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-h-[90vh] min-w-[90vw] overflow-auto lg:min-w-250">
		<Dialog.Header>
			<Dialog.Title>{m.add_api()}</Dialog.Title>
			<Dialog.Description>{m.select_an_api_this_client_should_access()}</Dialog.Description>
		</Dialog.Header>

		<AdvancedTable
			id="client-api-selection"
			{fetchCallback}
			onRowClick={(item) => onSelect(item)}
			defaultSort={{ column: 'name', direction: 'asc' }}
			{columns}
		/>
	</Dialog.Content>
</Dialog.Root>

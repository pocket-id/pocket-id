<script lang="ts">
	import { goto } from '$app/navigation';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import ApisService from '#lib/services/apis-service.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { Api } from '#lib/types/api.type.ts';
	import { LucidePencil } from '@lucide/svelte';
	import { deleteApiAction } from './api-actions';

	const apisService = new ApisService();
	let tableRef: AdvancedTable<Api>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<Api>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.api_resource(), column: 'resource', sortable: true },
		{ label: m.api_permissions(), key: 'permissions', value: (item) => item.permissions.length },
		{
			label: m.metadata_document_client_access(),
			column: 'allowCimdClients',
			value: (item) => (item.allowCimdClients ? m.enabled() : m.disabled())
		}
	];

	const actions: CreateAdvancedTableActions<Api> = () => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			variant: 'ghost',
			onClick: (api) => goto(`/settings/admin/apis/${api.id}`)
		},
		deleteApiAction(refresh)
	];
</script>

<AdvancedTable
	id="api-list"
	bind:this={tableRef}
	fetchCallback={apisService.list}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{actions}
/>

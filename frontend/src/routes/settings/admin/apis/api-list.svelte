<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { m } from '$lib/paraglide/messages';
	import ApisService from '$lib/services/apis-service';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { ApiListItem } from '$lib/types/api.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	const apisService = new ApisService();
	let tableRef: AdvancedTable<ApiListItem>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<ApiListItem>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.api_resource(), column: 'resource', sortable: true },
		{ label: m.api_permissions(), key: 'permissionCount', value: (item) => item.permissionCount }
	];

	const actions: CreateAdvancedTableActions<ApiListItem> = (api) => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			variant: 'ghost',
			onClick: (api) => goto(`/settings/admin/apis/${api.id}`)
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (api) => deleteApi(api)
		}
	];

	async function deleteApi(api: ApiListItem) {
		openConfirmDialog({
			title: m.delete_name({ name: api.name }),
			message: m.are_you_sure_you_want_to_delete_this_api(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await apisService.remove(api.id);
						await refresh();
						toast.success(m.api_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

<AdvancedTable
	id="api-list"
	bind:this={tableRef}
	fetchCallback={apisService.list}
	defaultSort={{ column: 'name', direction: 'asc' }}
	{columns}
	{actions}
/>

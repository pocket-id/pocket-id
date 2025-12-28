<script lang="ts">
	import { goto } from '$app/navigation';
	import { openConfirmDialog } from '$lib/components/confirm-dialog/';
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge/index';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import appConfigStore from '$lib/stores/application-configuration-store';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '$lib/types/advanced-table.type';
	import type { UserGroupMinimal } from '$lib/types/user-group.type';
	import { axiosErrorToast } from '$lib/utils/error-util';
	import { LucidePencil, LucideTrash } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	const userGroupService = new UserGroupService();
	let tableRef: AdvancedTable<UserGroupMinimal>;

	export function refresh() {
		return tableRef?.refresh();
	}

	const columns: AdvancedTableColumn<UserGroupMinimal>[] = [
		{ label: 'ID', column: 'id', hidden: true },
		{ label: m.friendly_name(), column: 'friendlyName', sortable: true },
		{ label: m.name(), column: 'name', sortable: true },
		{ label: m.user_count(), column: 'userCount', sortable: true },
		{
			label: m.created(),
			column: 'createdAt',
			sortable: true,
			hidden: true,
			value: (item) => new Date(item.createdAt).toLocaleString()
		},
		{ label: m.ldap_id(), column: 'ldapId', hidden: true },
		{ label: m.source(), key: 'source', hidden: !$appConfigStore.ldapEnabled, cell: SourceCell }
	];

	const actions: CreateAdvancedTableActions<UserGroupMinimal> = (group) => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			variant: 'ghost',
			onClick: (group) => goto(`/settings/admin/user-groups/${group.id}`)
		},
		{
			label: m.delete(),
			icon: LucideTrash,
			variant: 'danger',
			onClick: (group) => deleteUserGroup(group),
			visible: group.ldapId || $appConfigStore.ldapEnabled
		}
	];

	async function deleteUserGroup(userGroup: UserGroupMinimal) {
		openConfirmDialog({
			title: m.delete_name({ name: userGroup.name }),
			message: m.are_you_sure_you_want_to_delete_this_user_group(),
			confirm: {
				label: m.delete(),
				destructive: true,
				action: async () => {
					try {
						await userGroupService.remove(userGroup.id);
						await refresh();
						toast.success(m.user_group_deleted_successfully());
					} catch (e) {
						axiosErrorToast(e);
					}
				}
			}
		});
	}
</script>

{#snippet SourceCell({ item }: { item: UserGroupMinimal })}
	<Badge class="rounded-full" variant={item.ldapId ? 'default' : 'outline'}>
		{item.ldapId ? m.ldap() : m.local()}
	</Badge>
{/snippet}

<AdvancedTable
	id="user-group-list"
	bind:this={tableRef}
	fetchCallback={userGroupService.list}
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
	{columns}
	{actions}
/>

<script lang="ts">
	import { goto } from '$app/navigation';
	import AdvancedTable from '#lib/components/table/advanced-table.svelte';
	import { Badge } from '#lib/components/ui/badge/index.ts';
	import { m } from '#lib/paraglide/messages.js';
	import UserGroupService from '#lib/services/user-group-service.ts';
	import appConfigStore from '#lib/stores/application-configuration-store.ts';
	import type {
		AdvancedTableColumn,
		CreateAdvancedTableActions
	} from '#lib/types/advanced-table.type.ts';
	import type { UserGroupMinimal } from '#lib/types/user-group.type.ts';
	import { LucidePencil } from '@lucide/svelte';
	import { deleteUserGroupAction } from './user-group-actions';

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

	const actions: CreateAdvancedTableActions<UserGroupMinimal> = () => [
		{
			label: m.edit(),
			primary: true,
			icon: LucidePencil,
			variant: 'ghost',
			onClick: (group) => goto(`/settings/admin/user-groups/${group.id}`)
		},
		deleteUserGroupAction(refresh)
	];
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

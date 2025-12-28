<script lang="ts">
	import AdvancedTable from '$lib/components/table/advanced-table.svelte';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import type { AdvancedTableColumn } from '$lib/types/advanced-table.type';
	import type { UserGroupMinimal } from '$lib/types/user-group.type';

	let {
		selectionDisabled = false,
		selectedGroupIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

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
		{ label: m.ldap_id(), column: 'ldapId', hidden: true }
	];
</script>

<AdvancedTable
	id="user-group-selection"
	fetchCallback={userGroupService.list}
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
	bind:selectedIds={selectedGroupIds}
	{selectionDisabled}
	{columns}
/>

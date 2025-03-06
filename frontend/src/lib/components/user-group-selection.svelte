<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import UserGroupService from '$lib/services/user-group-service';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { UserGroup } from '$lib/types/user-group.type';

	let {
		groups: initialGroups,
		selectionDisabled = false,
		selectedGroupIds = $bindable()
	}: {
		groups: Paginated<UserGroup>;
		selectionDisabled?: boolean;
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

	let groups = $state(initialGroups);
</script>

<AdvancedTable
	items={groups}
	onRefresh={async (o) => (groups = await userGroupService.list(o))}
	columns={[{ label: 'Name', sortColumn: 'name' }]}
	bind:selectedIds={selectedGroupIds}
	{selectionDisabled}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.name}</Table.Cell>
	{/snippet}
</AdvancedTable>

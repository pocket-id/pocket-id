<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';

	let {
		selectionDisabled = false,
		selectedGroupIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();
</script>

<AdvancedTable
	id="user-group-selection"
	fetchCallback={userGroupService.list}
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
	columns={[{ label: m.name(), sortColumn: 'friendlyName' }]}
	bind:selectedIds={selectedGroupIds}
	{selectionDisabled}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.friendlyName}</Table.Cell>
	{/snippet}
</AdvancedTable>

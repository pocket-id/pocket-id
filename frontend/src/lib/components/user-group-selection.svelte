<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import UserGroupService from '$lib/services/user-group-service';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { UserGroup } from '$lib/types/user-group.type';
	import { onMount } from 'svelte';

	let {
		selectionDisabled = false,
		selectedGroupIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

	let groups: Paginated<UserGroup> | undefined = $state();
	let requestOptions: SearchPaginationSortRequest = $state({
		sort: {
			column: 'friendlyName',
			direction: 'asc'
		}
	});

	onMount(async () => {
		groups = await userGroupService.list(requestOptions);
	});
</script>

{#if groups}
	<AdvancedTable
		items={groups}
		{requestOptions}
		onRefresh={async (o) => (groups = await userGroupService.list(o))}
		columns={[{ label: m.name(), sortColumn: 'friendlyName' }]}
		bind:selectedIds={selectedGroupIds}
		{selectionDisabled}
	>
		{#snippet rows({ item })}
			<Table.Cell>{item.friendlyName}</Table.Cell>
		{/snippet}
	</AdvancedTable>
{/if}

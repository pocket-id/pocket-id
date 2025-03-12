<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import UserService from '$lib/services/user-service';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { User } from '$lib/types/user.type';
	import { onMount } from 'svelte';

	let {
		selectionDisabled = false,
		selectedUserIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedUserIds: string[];
	} = $props();

	const userService = new UserService();

	let users: Paginated<User> | undefined = $state();
	let requestOptions: SearchPaginationSortRequest = $state({
		sort: {
			column: 'firstName',
			direction: 'asc'
		}
	});

	onMount(async () => {
		users = await userService.list(requestOptions);
	});
</script>

{#if users}
	<AdvancedTable
		items={users}
		onRefresh={async (o) => (users = await userService.list(o))}
		{requestOptions}
		columns={[
			{ label: 'Name', sortColumn: 'firstName' },
			{ label: 'Email', sortColumn: 'email' }
		]}
		bind:selectedIds={selectedUserIds}
		{selectionDisabled}
	>
		{#snippet rows({ item })}
			<Table.Cell>{item.firstName} {item.lastName}</Table.Cell>
			<Table.Cell>{item.email}</Table.Cell>
		{/snippet}
	</AdvancedTable>
{/if}

<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import UserService from '$lib/services/user-service';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { User } from '$lib/types/user.type';
	import { onMount } from 'svelte';

	let {
		users: initialUsers,
		selectionDisabled = false,
		selectedUserIds = $bindable()
	}: { users: Paginated<User>; selectionDisabled?: boolean; selectedUserIds: string[] } = $props();
	let requestOptions: SearchPaginationSortRequest | undefined = $state({
		sort: { column: 'name', direction: 'asc' }
	});

	let users = $state<Paginated<User>>(initialUsers);

	// Fetch the sorted data when the component is mounted
	onMount(async () => {
		users = await userService.list(requestOptions!);
	});

	const userService = new UserService();
</script>

<AdvancedTable
	items={users}
	onRefresh={async (o) => (users = await userService.list(o))}
	{requestOptions}
	defaultSort={{ column: 'name', direction: 'asc' }}
	columns={[
		{ label: 'Name', sortColumn: 'name' },
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

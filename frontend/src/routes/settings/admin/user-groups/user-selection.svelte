<script lang="ts">
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
	import { m } from '$lib/paraglide/messages';
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

{#snippet FullNameCell({ item }: { item: User })}
	{item.firstName} {item.lastName}
{/snippet}

{#if users}
	<PocketIdTable
		items={users}
		bind:requestOptions
		bind:selectedIds={selectedUserIds}
		onRefresh={async (o) => (users = await userService.list(o))}
		columns={[
			{ title: m.name(), accessorKey: 'firstName', sortable: true, cell: FullNameCell },
			{ title: m.email(), accessorKey: 'email', sortable: true }
		]}
		persistKey="pocket-id-user-selection"
		{selectionDisabled}
	/>
{/if}

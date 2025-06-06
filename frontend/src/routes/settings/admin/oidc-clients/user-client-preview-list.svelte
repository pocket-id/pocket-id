<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { User } from '$lib/types/user.type';
	import { onMount } from 'svelte';
	import { Eye } from '@lucide/svelte';

	let {
		onPreview
	}: {
		onPreview: (userId: string) => void;
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

	function handlePreview(userId: string) {
		onPreview(userId);
	}
</script>

{#if users}
	<AdvancedTable
		items={users}
		onRefresh={async (o) => (users = await userService.list(o))}
		{requestOptions}
		columns={[
			{ label: m.name(), sortColumn: 'firstName' },
			{ label: m.email(), sortColumn: 'email' },
			{ label: m.actions() }
		]}
	>
		{#snippet rows({ item })}
			<Table.Cell>{item.firstName} {item.lastName}</Table.Cell>
			<Table.Cell>{item.email}</Table.Cell>
			<Table.Cell>
				<Button size="sm" variant="outline" onclick={() => handlePreview(item.id)} class="gap-2">
					<Eye class="h-4 w-4" />
					{m.preview()}
				</Button>
			</Table.Cell>
		{/snippet}
	</AdvancedTable>
{/if}

<script lang="ts">
	import PocketIdTable from '$lib/components/pocket-id-table/pocket-id-table.svelte';
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
	<PocketIdTable
		items={groups}
		bind:requestOptions
		bind:selectedIds={selectedGroupIds}
		onRefresh={async (opts) => (groups = await userGroupService.list(opts))}
		columns={[{ title: m.name(), accessorKey: 'friendlyName', sortable: true }]}
		persistKey="pocket-id-group-selection"
		{selectionDisabled}
	/>
{/if}

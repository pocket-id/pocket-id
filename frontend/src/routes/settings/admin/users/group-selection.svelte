<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import UserGroupService from '$lib/services/user-group-service';
	import type { Paginated } from '$lib/types/pagination.type';
	import type { SearchPaginationSortRequest } from '$lib/types/pagination.type';
	import type { UserGroupWithUserCount } from '$lib/types/user-group.type';
	import Badge from '$lib/components/ui/badge/badge.svelte'; // Import Badge directly
	import appConfigStore from '$lib/stores/application-configuration-store';
	import { onMount } from 'svelte';

	let {
		groups: initialGroups,
		selectionDisabled = false,
		selectedGroupIds = $bindable()
	}: {
		groups: Paginated<UserGroupWithUserCount>;
		selectionDisabled?: boolean;
		selectedGroupIds: string[];
	} = $props();

	const userGroupService = new UserGroupService();

	// Initialize groups and requestOptions properly
	let groups = $state(initialGroups);

	// Initialize requestOptions with proper defaults
	let requestOptions = $state<SearchPaginationSortRequest>({
		search: '',
		sort: { column: 'friendlyName', direction: 'asc' },
		pagination: {
			page: initialGroups.pagination?.currentPage || 1,
			limit: initialGroups.pagination?.itemsPerPage || 10
		}
	});

	// Helper function to update the groups state with the new data
	function updateGroups(newGroups: Paginated<UserGroupWithUserCount>) {
		groups = newGroups;
		return groups;
	}

	onMount(async () => {
		try {
			// Use the default sort options to fetch correctly sorted data
			const sortedGroups = await userGroupService.list({
				search: '',
				sort: { column: 'friendlyName', direction: 'asc' },
				pagination: {
					page: initialGroups.pagination?.currentPage || 1,
					limit: initialGroups.pagination?.itemsPerPage || 10
				}
			});
			updateGroups(sortedGroups);
		} catch (error) {
			console.error('Error performing initial sort:', error);
		}
	});
</script>

<AdvancedTable
	items={groups}
	bind:requestOptions
	onRefresh={async (options) => {
		try {
			// Create a fresh copy of options to avoid reactivity issues
			const normalizedOptions = {
				search: options.search || '',
				sort: options.sort || { column: 'friendlyName', direction: 'asc' },
				pagination: options.pagination || { page: 1, limit: 10 }
			};

			// Update local state
			requestOptions = normalizedOptions;

			// Fetch data and update groups
			const refreshedGroups = await userGroupService.list(normalizedOptions);
			return updateGroups(refreshedGroups);
		} catch (error) {
			console.error('Error refreshing groups:', error);
			return groups; // Return current state on error
		}
	}}
	columns={[
		{ label: 'Friendly Name', sortColumn: 'friendlyName' },
		{ label: 'Name', sortColumn: 'name' },
		{ label: 'User Count', sortColumn: 'userCount' },
		...($appConfigStore.ldapEnabled ? [{ label: 'Source' }] : [])
	]}
	bind:selectedIds={selectedGroupIds}
	{selectionDisabled}
	defaultSort={{ column: 'friendlyName', direction: 'asc' }}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.friendlyName}</Table.Cell>
		<Table.Cell>{item.name}</Table.Cell>
		<Table.Cell>{item.userCount}</Table.Cell>
		{#if $appConfigStore.ldapEnabled}
			<Table.Cell>
				<Badge variant={item.ldapId ? 'default' : 'outline'}>
					{item.ldapId ? 'LDAP' : 'Local'}
				</Badge>
			</Table.Cell>
		{/if}
	{/snippet}
</AdvancedTable>

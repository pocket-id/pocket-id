<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import UserService from '$lib/services/user-service';

	let {
		selectionDisabled = false,
		selectedUserIds = $bindable()
	}: {
		selectionDisabled?: boolean;
		selectedUserIds: string[];
	} = $props();

	const userService = new UserService();
</script>

<AdvancedTable
	id="user-selection"
	fetchCallback={userService.list}
	defaultSort={{ column: 'firstName', direction: 'asc' }}
	columns={[
		{ label: m.name(), sortColumn: 'firstName' },
		{ label: m.email(), sortColumn: 'email' }
	]}
	bind:selectedIds={selectedUserIds}
	{selectionDisabled}
>
	{#snippet rows({ item })}
		<Table.Cell>{item.firstName} {item.lastName}</Table.Cell>
		<Table.Cell>{item.email}</Table.Cell>
	{/snippet}
</AdvancedTable>

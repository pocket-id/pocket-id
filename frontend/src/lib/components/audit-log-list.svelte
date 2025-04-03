<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as Table from '$lib/components/ui/table';
	import { m } from '$lib/paraglide/messages';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';

	let {
		auditLogs,
		isAdmin = false,
		requestOptions
	}: {
		auditLogs: Paginated<AuditLog>;
		isAdmin?: boolean;
		requestOptions: SearchPaginationSortRequest;
	} = $props();

	const auditLogService = new AuditLogService();

	function toFriendlyEventString(event: string) {
		const words = event.split('_');
		const capitalizedWords = words.map((word) => {
			return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
		});
		return capitalizedWords.join(' ');
	}
</script>

<AdvancedTable
	items={auditLogs}
	{requestOptions}
	onRefresh={async (options) =>
		isAdmin
			? (auditLogs = await auditLogService.listAllLogs(options))
			: (auditLogs = await auditLogService.list(options))}
	columns={[
		{ label: m.time(), sortColumn: 'createdAt' },
        ...(isAdmin ? [{ label: 'Username' }] : []),
		{ label: m.event(), sortColumn: 'event' },
		{ label: m.approximate_location(), sortColumn: 'city' },
		{ label: m.ip_address(), sortColumn: 'ipAddress' },
		{ label: m.device(), sortColumn: 'device' },
		{ label: m.client() }
	]}
	withoutSearch
>
	{#snippet rows({ item })}
		<Table.Cell>{new Date(item.createdAt).toLocaleString()}</Table.Cell>
		{#if isAdmin}
			<Table.Cell>
				{#if item.username}
					{item.username}
				{:else}
					Unknown User
				{/if}
			</Table.Cell>
		{/if}
		<Table.Cell>
			<Badge variant="outline">{toFriendlyEventString(item.event)}</Badge>
		</Table.Cell>
		<Table.Cell
			>{item.city && item.country ? `${item.city}, ${item.country}` : m.unknown()}</Table.Cell
		>
		<Table.Cell>{item.ipAddress}</Table.Cell>
		<Table.Cell>{item.device}</Table.Cell>
		<Table.Cell>{item.data.clientName}</Table.Cell>
	{/snippet}
</AdvancedTable>

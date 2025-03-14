<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as Table from '$lib/components/ui/table';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';

	let {
		auditLogs,
		requestOptions
	}: { auditLogs: Paginated<AuditLog>; requestOptions: SearchPaginationSortRequest } = $props();

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
	onRefresh={async (options) => (auditLogs = await auditLogService.list(options))}
	columns={[
		{ label: 'Time', sortColumn: 'createdAt' },
		{ label: 'Event', sortColumn: 'event' },
		{ label: 'Approximate Location', sortColumn: 'city' },
		{ label: 'IP Address', sortColumn: 'ipAddress' },
		{ label: 'Device', sortColumn: 'device' },
		{ label: 'Client' }
	]}
	withoutSearch
>
	{#snippet rows({ item })}
		<Table.Cell>{new Date(item.createdAt).toLocaleString()}</Table.Cell>
		<Table.Cell>
			<Badge variant="outline">{toFriendlyEventString(item.event)}</Badge>
		</Table.Cell>
		<Table.Cell
			>{item.city && item.country ? `${item.city}, ${item.country}` : 'Unknown'}</Table.Cell
		>
		<Table.Cell>{item.ipAddress}</Table.Cell>
		<Table.Cell>{item.device}</Table.Cell>
		<Table.Cell>{item.data.clientName}</Table.Cell>
	{/snippet}
</AdvancedTable>

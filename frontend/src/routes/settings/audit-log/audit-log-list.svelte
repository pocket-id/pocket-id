<script lang="ts">
	import AdvancedTable from '$lib/components/advanced-table.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import * as Table from '$lib/components/ui/table';
	import AuditLogService from '$lib/services/audit-log-service';
	import type { AuditLog } from '$lib/types/audit-log.type';
	import type { Paginated, SearchPaginationSortRequest } from '$lib/types/pagination.type';

	let {
		auditLogs,
    isAdmin = flase;
		requestOptions
	}: { auditLogs: Paginated<AuditLog>; isAdmin?: boolean; requestOptions: SearchPaginationSortRequest } = $props();

	const auditLogService = new AuditLogService();

	function toFriendlyEventString(event: string) {
		const words = event.split('_');
		const capitalizedWords = words.map((word) => {
			return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
		});
		return capitalizedWords.join(' ');
	}

	// Expose this function for parent components
	export async function refreshAuditLogs(options: any) {
		// Extract filters from options if they exist
		const filters = options.filters || {};
		
		// Create params with the expected structure for backend
		const params: any = {
			sort: options.sort,
			pagination: options.pagination
		};

		// Only add filters if there are any
		if (Object.keys(filters).length > 0) {
			// Initialize filters property
			params.filters = {};
			
			// Add each filter to params.filters, extracting value from proxy objects if needed
			if (filters.userId) {
				params.filters.userId = typeof filters.userId === 'object' && 'value' in filters.userId 
					? filters.userId.value 
					: filters.userId;
			}

			if (filters.event) {
				params.filters.event = typeof filters.event === 'object' && 'value' in filters.event 
					? filters.event.value 
					: filters.event;
			}

			if (filters.clientId) {
				params.filters.clientId = typeof filters.clientId === 'object' && 'value' in filters.clientId 
					? filters.clientId.value 
					: filters.clientId;
			}
		}

		console.log('Final params:', params); // Debug only

		// Call the appropriate API endpoint
		if (isAdmin) {
			auditLogs = await auditLogService.listAllLogs(params);
		} else {
			auditLogs = await auditLogService.list(params);
		}

		return auditLogs;
	}
</script>

<AdvancedTable
	items={auditLogs}
	{requestOptions}
	onRefresh={async (options) => (auditLogs = await auditLogService.list(options))}
	columns={[
		{ label: 'Time', sortColumn: 'createdAt' },
		...(isAdmin ? [{ label: 'Username' }] : []),
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
			>{item.city && item.country ? `${item.city}, ${item.country}` : 'Unknown'}</Table.Cell
		>
		<Table.Cell>{item.ipAddress}</Table.Cell>
		<Table.Cell>{item.device}</Table.Cell>
		<Table.Cell>{item.data.clientName}</Table.Cell>
	{/snippet}
</AdvancedTable>
